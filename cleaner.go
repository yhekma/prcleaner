package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os/exec"
	"strings"
)

type Hook struct {
	// branch action
	Ref        string `json:"ref"`
	Deleted    bool   `json:"deleted"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`

	// pr action
	Action      string `json:"action"` // either "opened" or "closed" if pr
	Number      int    `json:"number"`
	PullRequest struct {
		Head struct {
			Sha string `json:"sha"`
			Ref string `json:"ref"`
		} `json:"head"`
	} `json:"pull_request"`
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	var pr Hook
	err := json.NewDecoder(r.Body).Decode(&pr)
	CheckErr(err)
	_, _ = fmt.Fprint(w, http.StatusAccepted)
	var selector string

	if pr.Action == "closed" {
		selector = fmt.Sprintf(
			"%s=PR-%d,%s=%s,%s=%s", *BranchLabel, pr.Number, *RepoLabel, pr.Repository.Name, *CommitShaLabel, pr.PullRequest.Head.Sha,
		)
	}

	if pr.Action == "opened" {
		selector = fmt.Sprintf(
			"%s=%s,%s=%s,%s=%s", *BranchLabel, pr.PullRequest.Head.Ref, *RepoLabel, pr.Repository.Name, *CommitShaLabel, pr.PullRequest.Head.Sha,
		)
	}

	if pr.Deleted {
		branchName := strings.Split(pr.Ref, "/")[2]
		selector = fmt.Sprintf(
			"%s=%s,%s=%s,%s", *BranchLabel, branchName, *RepoLabel, pr.Repository.Name, *CommitShaLabel)
	}

	log.Debug("using selector ", selector)

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	pods, err := Clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	CheckErr(err)
	for _, pod := range pods.Items {
		// We only want pods in the namespace "app" or "app-(.*)"
		release := pod.Labels[*ReleaseLabel]
		if release == "" {
			continue
		}

		log.WithFields(log.Fields{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
			"pr":        pr.Number,
		}).Debug("found pod to delete")

		var out []byte
		if *Dryrun {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release, "--dry-run").Output()
		} else {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release).Output()
		}
		CheckErr(err)
		log.WithFields(log.Fields{
			"helm command": fmt.Sprintf("/bin/helm uninstall -n %s %s --dry-run", pod.Namespace, release),
			"output":       string(out),
		}).Debug()
	}
}

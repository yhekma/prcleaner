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
	Before     string `json:"before"` // This is sha
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

func cleaner(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		_, _ = fmt.Fprintf(w, "400")
	}
	var hook Hook
	err := json.NewDecoder(r.Body).Decode(&hook)
	log.WithFields(log.Fields{
		"hook content": fmt.Sprintf("%+v", hook),
	}).Debug("decoded hook")
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(w, http.StatusAccepted)
	var selector string

	if hook.Action == "closed" {
		selector = fmt.Sprintf(
			"%s=PR-%d,%s=%s,%s=%s", C.BranchLabel, hook.Number, C.RepoLabel, hook.Repository.Name, C.CommitShaLabel, hook.PullRequest.Head.Sha,
		)
	}

	if hook.Action == "opened" || hook.Action == "reopened" {
		selector = fmt.Sprintf(
			"%s=%s,%s=%s,%s=%s", C.BranchLabel, hook.PullRequest.Head.Ref, C.RepoLabel, hook.Repository.Name, C.CommitShaLabel, hook.PullRequest.Head.Sha,
		)
	}

	if hook.Deleted { // Branch deletion
		branchName := strings.Split(hook.Ref, "/")[2]
		selector = fmt.Sprintf(
			"%s=%s,%s=%s,%s=%s", C.BranchLabel, branchName, C.RepoLabel, hook.Repository.Name, C.CommitShaLabel, hook.Before)
	}

	if selector == "" {
		log.Debug("no action needed")
		return nil
	}

	log.Debug("using selector ", selector)

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}

	pods, err := Clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		release := pod.Labels[C.ReleaseLabel]
		if release == "" {
			continue
		}

		log.WithFields(log.Fields{
			"pod":       pod.Name,
			"namespace": pod.Namespace,
			"hook":      hook.Number,
		}).Debug("found pod to delete")

		var out []byte
		if C.Dryrun {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release, "--dry-run").Output()
		} else {
			out, err = exec.Command("/bin/helm", "uninstall", "-n", pod.Namespace, release).Output()
		}
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"helm command": fmt.Sprintf("/bin/helm uninstall -n %s %s --dry-run", pod.Namespace, release),
			"output":       string(out),
		}).Debug()
	}
	return nil
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	err := cleaner(w, r)
	CheckErr(err)
}

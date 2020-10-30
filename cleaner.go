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

type Pr struct {
	Action string `json:"action"`
	Number int `json:"number"`
	PullRequest struct {
		Url string `json:"url"`
		State string `json:"state"`
		Head struct {
			Repo struct {
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
}

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	var pr Pr
	err := json.NewDecoder(r.Body).Decode(&pr)
	CheckErr(err)
	_, _ = fmt.Fprint(w, http.StatusAccepted)

	splittedName := strings.Split(pr.PullRequest.Head.Repo.FullName, "/")
	prOrg := splittedName[0]
	prRepo := splittedName[1]
	selector := fmt.Sprintf("%s=PR-%d,%s=%s", *BranchLabel, pr.Number, *RepoLabel, prRepo)
	log.Debug("using selector ", selector)
	if *Org != prOrg {
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}
	pods, err := Clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	CheckErr(err)
	for _, pod := range pods.Items {
		// We only want pods in the namespace "app" or "app-(.*)"
		if ! strings.HasPrefix(pod.Namespace, fmt.Sprintf("%s-", *App)) || pod.Namespace != *App {
			release := pod.Labels[*ReleaseLabel]
			if release == "" {
				continue
			}

			log.WithFields(log.Fields{
				"pod": pod.Name,
				"namespace": pod.Namespace,
				"pr": pr.Number,
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
				"output": string(out),
			}).Debug()
		}
	}
}
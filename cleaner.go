package main

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os/exec"
	"strings"
)

func cleaner(w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		_, _ = fmt.Fprintf(w, "400")
	}
	payload, err := github.ValidatePayload(r, []byte(C.Secret))
	if err != nil {
		return err
	}
	defer r.Body.Close()

	hook, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprint(w, http.StatusAccepted)
	var selector string

	switch e := hook.(type) {
	case *github.PullRequestEvent:
		log.Debugf("received %+v", e)
		if *e.Action == "closed" {
			selector = fmt.Sprintf("%s=PR-%d,%s=%s,%s=%s", C.BranchLabel, e.Number, C.RepoLabel, *e.Repo.Name, C.CommitShaLabel, *e.PullRequest.Head.SHA)
		}
		if *e.Action == "opened" || *e.Action == "reopened" {
			selector = fmt.Sprintf(
				"%s=%s,%s=%s,%s=%s", C.BranchLabel, *e.PullRequest.Head.Ref, C.RepoLabel, *e.Repo.Name, C.CommitShaLabel, *e.PullRequest.Head.SHA,
			)
		}
	case *github.PushEvent:
		log.Debugf("received %+v", e)
		if *e.Deleted {
			branchName := strings.Split(*e.Ref, "/")[2]
			selector = fmt.Sprintf(
				"%s=%s,%s=%s,%s=%s", C.BranchLabel, branchName, C.RepoLabel, *e.Repo.Name, C.CommitShaLabel, *e.Before,
			)
		}
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

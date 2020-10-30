package main

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

func CleanerServer(w http.ResponseWriter, r *http.Request) {
	var pr Pr
	err := json.NewDecoder(r.Body).Decode(&pr)
	CheckErr(err)
	splittedName := strings.Split(pr.PullRequest.Head.Repo.FullName, "/")
	prOrg := splittedName[0]
	prRepo := splittedName[1]
	selector := fmt.Sprintf("app.fedex.io/git-branch=PR-%d,app.fedex.io/git-repository=%s", pr.Number, prRepo)
	log.Debug("using selector ", selector)
	if Org != prOrg {
		return
	}

	listOptions := metav1.ListOptions{
		LabelSelector: selector,
	}
	pods, err := Clientset.CoreV1().Pods("").List(context.TODO(), listOptions)
	CheckErr(err)
	for _, pod := range pods.Items {
		if ! strings.HasPrefix(pod.Namespace, fmt.Sprintf("%s-", App)) || pod.Namespace != App {
			log.WithFields(log.Fields{
				"pod": pod.Name,
				"namespace": pod.Namespace,
				"pr": pr.Number,
			}).Debug("delete pod")
		}
	}
}
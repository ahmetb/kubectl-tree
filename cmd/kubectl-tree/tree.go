package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"
	"sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

const (
	firstElemPrefix = `├─`
	lastElemPrefix  = `└─`
	indent          = "  "
	pipe            = `│ `
)

var (
	gray   = color.New(color.FgHiBlack)
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
)

// treeView prints object hierarchy to out stream.
func treeView(out io.Writer, objs objectDirectory, obj unstructured.Unstructured, conditionTypes []string) {
	tbl := uitable.New()
	tbl.Separator = "  "
	tbl.AddRow("NAMESPACE", "NAME", "READY", "REASON", "STATUS", "AGE")
	treeViewInner("", tbl, objs, obj, conditionTypes)
	fmt.Fprintln(color.Output, tbl)
}

func treeViewInner(prefix string, tbl *uitable.Table, objs objectDirectory, obj unstructured.Unstructured, conditionTypes []string) {
	ready, reason, kstatus := extractStatus(obj, conditionTypes)

	var readyColor *color.Color
	switch ready {
	case "True":
		readyColor = green
	case "False", "Unknown":
		readyColor = red
	default:
		readyColor = gray
	}
	if ready == "" {
		ready = "-"
	}

	var statusColor *color.Color
	switch kstatus {
	case status.CurrentStatus:
		statusColor = green
	case status.InProgressStatus:
		statusColor = yellow
	case status.FailedStatus, status.TerminatingStatus:
		statusColor = red
	default:
		statusColor = gray
	}
	if kstatus == "" {
		kstatus = "-"
	}

	c := obj.GetCreationTimestamp()
	age := duration.HumanDuration(time.Since(c.Time))
	if c.IsZero() {
		age = "<unknown>"
	}

	tbl.AddRow(obj.GetNamespace(), fmt.Sprintf("%s%s/%s",
		gray.Sprint(printPrefix(prefix)),
		obj.GetKind(),
		color.New(color.Bold).Sprint(obj.GetName())),
		readyColor.Sprint(ready),
		readyColor.Sprint(reason),
		statusColor.Sprint(kstatus),
		age)
	chs := objs.ownedBy(obj.GetUID())
	for i, child := range chs {
		var p string
		switch i {
		case len(chs) - 1:
			p = prefix + lastElemPrefix
		default:
			p = prefix + firstElemPrefix
		}
		treeViewInner(p, tbl, objs, child, conditionTypes)
	}
}

func printPrefix(p string) string {
	// this part is hacky af
	if strings.HasSuffix(p, firstElemPrefix) {
		p = strings.Replace(p, firstElemPrefix, pipe, strings.Count(p, firstElemPrefix)-1)
	} else {
		p = strings.ReplaceAll(p, firstElemPrefix, pipe)
	}

	if strings.HasSuffix(p, lastElemPrefix) {
		p = strings.Replace(p, lastElemPrefix, strings.Repeat(" ", len([]rune(lastElemPrefix))), strings.Count(p, lastElemPrefix)-1)
	} else {
		p = strings.ReplaceAll(p, lastElemPrefix, strings.Repeat(" ", len([]rune(lastElemPrefix))))
	}
	return p
}

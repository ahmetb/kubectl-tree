package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/gosuri/uitable"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

const (
	firstElemPrefix = `├─`
	lastElemPrefix  = `└─`
	indent          = "  "
	pipe            = `│ `
)

var (
	gray  = color.New(color.FgHiBlack)
	red   = color.New(color.FgRed)
	green = color.New(color.FgGreen)
)

// treeView prints object hierarchy to out stream.
func treeView(out io.Writer, objs objectDirectory, obj unstructured.Unstructured) {
	tbl := uitable.New()
	tbl.Separator = "  "
	tbl.AddRow("NAMESPACE", "NAME", "READY", "REASON")
	treeViewInner("", tbl, objs, obj)
	fmt.Fprintln(out, tbl)
}

func treeViewInner(prefix string, tbl *uitable.Table, objs objectDirectory, obj unstructured.Unstructured) {
	ready, reason := extractStatus(obj)

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

	tbl.AddRow(obj.GetNamespace(), fmt.Sprintf("%s%s/%s",
		gray.Sprint(printPrefix(prefix)),
		obj.GetKind(),
		color.New(color.Bold).Sprint(obj.GetName())),
		readyColor.Sprint(ready),
		readyColor.Sprint(reason))
	chs := objs.ownedBy(obj.GetUID())
	for i, child := range chs {
		var p string
		switch i {
		case len(chs) - 1:
			p = prefix + lastElemPrefix
		default:
			p = prefix + firstElemPrefix
		}
		treeViewInner(p, tbl, objs, child)
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

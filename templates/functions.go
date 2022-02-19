// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package templates

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"
)

// NewContextFunc creates a new function that can be used to store
// and access arbitrary data by keys.
func NewContextFunc(m map[string]any) func(string) any {
	return func(key string) any {
		if value, ok := m[key]; ok {
			return value
		}
		return nil
	}
}

var defaultFunctions = template.FuncMap{
	"safehtml":        safeHTMLFunc,
	"relative_time":   relativeTimeFunc,
	"year_range":      yearRangeFunc,
	"contains_string": containsStringFunc,
	"html_br":         htmlBrFunc,
	"map":             mapFunc,
}

func safeHTMLFunc(text string) template.HTML {
	return template.HTML(text)
}

func relativeTimeFunc(t time.Time) string {
	const day = 24 * time.Hour
	d := time.Since(t)
	switch {
	case d < time.Second:
		return "just now"
	case d < 2*time.Second:
		return "one second ago"
	case d < time.Minute:
		return fmt.Sprintf("%d seconds ago", d/time.Second)
	case d < 2*time.Minute:
		return "one minute ago"
	case d < time.Hour:
		return fmt.Sprintf("%d minutes ago", d/time.Minute)
	case d < 2*time.Hour:
		return "one hour ago"
	case d < day:
		return fmt.Sprintf("%d hours ago", d/time.Hour)
	case d < 2*day:
		return "one day ago"
	case d < 30*day:
		return fmt.Sprintf("%d days ago", d/day)
	case d < 60*day:
		return "one month ago"
	case d < 2*365*day:
		return fmt.Sprintf("%d months ago", d/30/day)
	}
	return fmt.Sprintf("%d years ago", d/365/day)
}

func yearRangeFunc(year int) string {
	curYear := time.Now().Year()
	if year >= curYear {
		return fmt.Sprintf("%d", year)
	}
	return fmt.Sprintf("%d - %d", year, curYear)
}

func containsStringFunc(list []string, element, yes, no string) string {
	for _, e := range list {
		if e == element {
			return yes
		}
	}
	return no
}

func htmlBrFunc(text string) string {
	return strings.ReplaceAll(text, "\n", "<br>")
}

func mapFunc(values ...string) (map[string]string, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid map call")
	}
	m := make(map[string]string, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		m[values[i]] = values[i+1]
	}
	return m, nil
}

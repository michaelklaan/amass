// Copyright 2017-2021 Jeff Foley. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package enum

import (
	"time"

	"github.com/OWASP/Amass/v3/filter"
	"github.com/OWASP/Amass/v3/requests"
)

// ExtractOutput is a convenience method for obtaining new discoveries made by the enumeration process.
func (e *Enumeration) ExtractOutput(filter filter.Filter, asinfo bool) []*requests.Output {
	if e.Config.Passive {
		return e.Graph.EventNames(e.Config.UUID.String(), filter)
	}

	return e.Graph.EventOutput(e.Config.UUID.String(), filter, asinfo, e.Sys.Cache())
}

func (e *Enumeration) submitKnownNames() {
	filter := filter.NewStringFilter()

	for _, g := range e.Sys.GraphDatabases() {
		var events []string

		for _, event := range g.EventList() {
			for _, domain := range g.EventDomains(event) {
				if e.Config.IsDomainInScope(domain) {
					events = append(events, event)
					break
				}
			}
		}

		for _, event := range events {
			for _, output := range g.EventNames(event, filter) {
				select {
				case <-e.done:
					return
				default:
				}

				if e.Config.IsDomainInScope(output.Name) {
					e.nameSrc.InputName(&requests.DNSRequest{
						Name:   output.Name,
						Domain: output.Domain,
						Tag:    output.Tag,
						Source: output.Sources[0],
					})
				}
			}
		}
	}
}

func (e *Enumeration) submitProvidedNames() {
	for _, name := range e.Config.ProvidedNames {
		if domain := e.Config.WhichDomain(name); domain != "" {
			e.nameSrc.InputName(&requests.DNSRequest{
				Name:   name,
				Domain: domain,
				Tag:    requests.EXTERNAL,
				Source: "User Input",
			})
		}
	}
}

func (e *Enumeration) queueLog(msg string) {
	e.logQueue.Append(msg)
}

func (e *Enumeration) writeLogs(all bool) {
	num := e.logQueue.Len() / 10
	if num <= 1000 {
		num = 1000
	}

	for i := 0; ; i++ {
		msg, ok := e.logQueue.Next()
		if !ok {
			break
		}

		if e.Config.Log != nil {
			e.Config.Log.Print(msg.(string))
		}

		if !all && i >= num {
			break
		}
	}
}

func (e *Enumeration) periodicLogging() {
	t := time.NewTimer(5 * time.Second)

	for {
		select {
		case <-e.done:
			return
		case <-t.C:
			e.writeLogs(false)
			t.Reset(5 * time.Second)
		}
	}
}

package main

import (
	"github.com/abh/geodns/countries"
	"github.com/miekg/dns"
	"strings"
	"time"
)

type Options struct {
	Serial   int
	Ttl      int
	MaxHosts int
	Contact  string
}

type Record struct {
	RR     dns.RR
	Weight int
}

type Records []Record

func (s Records) Len() int      { return len(s) }
func (s Records) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type RecordsByWeight struct{ Records }

func (s RecordsByWeight) Less(i, j int) bool { return s.Records[i].Weight > s.Records[j].Weight }

type Label struct {
	Label    string
	MaxHosts int
	Ttl      int
	Records  map[uint16]Records
	Weight   map[uint16]int
}

type labels map[string]*Label

type Zones map[string]*Zone

type Zone struct {
	Origin    string
	Labels    labels
	LenLabels int
	Options   Options
	LastRead  time.Time
}

type qTypes []uint16

func (l *Label) firstRR(dnsType uint16) dns.RR {
	return l.Records[dnsType][0].RR
}

func (z *Zone) AddLabel(k string) *Label {
	k = strings.ToLower(k)
	z.Labels[k] = new(Label)
	label := z.Labels[k]
	label.Label = k
	label.Ttl = z.Options.Ttl
	label.MaxHosts = z.Options.MaxHosts

	label.Records = make(map[uint16]Records)
	label.Weight = make(map[uint16]int)

	return label
}

func (z *Zone) SoaRR() dns.RR {
	return z.Labels[""].firstRR(dns.TypeSOA)
}

func (z *Zone) findLabels(s, cc string, qts qTypes) (*Label, uint16) {

	selectors := []string{}

	if len(cc) > 0 {
		continent := countries.CountryContinent[cc]
		var s_cc string
		if len(s) > 0 {
			s_cc = s + "." + cc
			if len(continent) > 0 {
				continent = s + "." + continent
			}
		} else {
			s_cc = cc
		}
		selectors = append(selectors, s_cc, continent)
	}
	selectors = append(selectors, s)

	for _, name := range selectors {

		if label, ok := z.Labels[name]; ok {

			for _, qtype := range qts {

				switch qtype {
				case dns.TypeANY:
					// short-circuit mostly to avoid subtle bugs later
					// to be correct we should run through all the selectors and
					// pick types not already picked
					return z.Labels[s], qtype
				case dns.TypeMF:
					if label.Records[dns.TypeMF] != nil {
						name = label.firstRR(dns.TypeMF).(*dns.MF).Mf
						// TODO: need to avoid loops here somehow
						return z.findLabels(name, cc, qts)
					}
				default:
					// return the label if it has the right record
					if label.Records[qtype] != nil && len(label.Records[qtype]) > 0 {
						return label, qtype
					}
				}
			}
		}
	}

	return z.Labels[s], 0
}

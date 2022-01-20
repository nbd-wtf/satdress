package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type BaseData struct {
	Domains       []string `json:"domains"`
	SiteOwnerName string   `json:"siteOwnerName"`
	SiteOwnerURL  string   `json:"siteOwnerURL"`
	SiteName      string   `json:"siteName"`
	UsernameInfo  string   `json:"usernameInfo"`
}

func renderHTML(w http.ResponseWriter, html string, extraData interface{}) {
	info := "Desired Username"
	if s.GlobalUsers {
		info = "Desired Username (unique across all domains)"
	}
	base, _ := json.Marshal(BaseData{
		Domains:       getDomains(s.Domain),
		SiteOwnerName: s.SiteOwnerName,
		SiteOwnerURL:  s.SiteOwnerURL,
		SiteName:      s.SiteName,
		UsernameInfo:  info,
	})
	extra, _ := json.Marshal(extraData)

	w.Header().Set("content-type", "text/html")
	fmt.Fprint(w,
		strings.ReplaceAll(
			strings.ReplaceAll(
				html,
				"{} // REPLACED WITH SERVER DATA",
				fmt.Sprintf("{...%s, ...%s}", string(base), string(extra)),
			),
			"Satdress", s.SiteName,
		),
	)
}

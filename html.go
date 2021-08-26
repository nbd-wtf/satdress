package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type BaseData struct {
	Domain        string `json:"domain"`
	SiteOwnerName string `json:"siteOwnerName"`
	SiteOwnerURL  string `json:"siteOwnerURL"`
	SiteName      string `json:"siteName"`
}

func renderHTML(w http.ResponseWriter, html string, extraData interface{}) {
	base, _ := json.Marshal(BaseData{
		Domain:        s.Domain,
		SiteOwnerName: s.SiteOwnerName,
		SiteOwnerURL:  s.SiteOwnerURL,
		SiteName:      s.SiteName,
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

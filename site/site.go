/**
 * Created by lock
 * Date: 2019-08-12
 * Time: 11:36
 */
package site

import (
	"fmt"
	"gochat/config"
	"gochat/log"
	"net/http"
)

type Site struct {
}

func New() *Site {
	return &Site{}
}

func (s *Site) Run() {
	siteConfig := config.Conf.Site
	port := siteConfig.SiteBase.ListenPort
	addr := fmt.Sprintf(":%d", port)
	log.Log.Fatal(http.ListenAndServe(addr, http.FileServer(http.Dir("./site/"))))
}

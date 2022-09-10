// A binary that starts a local server, waits for a page to render,
// then downloads the source as HTML.
package main

import (
	"flag"
	"io/ioutil"
	"strings"
	"time"

	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spudtrooper/goutil/check"
	"github.com/spudtrooper/goutil/flags"
	"github.com/spudtrooper/goutil/or"
	"github.com/spudtrooper/goutil/request"
	goutilselenium "github.com/spudtrooper/goutil/selenium"
	"github.com/tebeka/selenium"
)

var (
	port            = flags.Int("port", "port to run the server on")
	dir             = flags.String("dir", "directory with static files")
	seleniumVerbose = flags.Bool("selnenium_verbose", "verbose selenium logging")
	seleniumHead    = flags.Bool("selnenium_head", "run in selenium head mode")
	page            = flags.String("page", "page to render")
	selector        = flags.String("selector", "CSS selector for which to wait")
	outfile         = flags.String("outfile", "file to which we write HTML")
	noserver        = flags.Bool("noserver", "don't start a server")
)

func startLocalServerr(ctx context.Context) error {
	port := or.Int(*port, 8084)

	http.Handle("/", http.FileServer(http.Dir(*dir)))
	log.Printf("Starting server on http://localhost:%d for %s", port, *dir)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		return err
	}
	return nil
}

func uri() string {
	return fmt.Sprintf("http://localhost:%d/%s", *port, *page)
}

func waitForRender() error {
	wd, cancel, err := goutilselenium.MakeWebDriver(goutilselenium.MakeWebDriverOptions{
		Verbose:  *seleniumVerbose,
		Headless: !*seleniumHead,
	})
	if err != nil {
		return err
	}
	defer cancel()

	if err := wd.Get(uri()); err != nil {
		return err
	}

	start := time.Now()
	wd.Wait(func(wd selenium.WebDriver) (bool, error) {
		log.Printf("waiting for element with selector %q... it has been %v", *selector, time.Since(start))
		el, err := wd.FindElement(selenium.ByCSSSelector, *selector)
		if err != nil {
			if strings.Contains(err.Error(), "no such element") {
				return false, nil
			}
			return false, err
		}
		if el != nil {
			return false, nil
		}
		return true, nil
	})
	log.Printf("done waiting")

	el, err := wd.FindElement(selenium.ByTagName, "html")
	if err != nil {
		return err
	}

	src, err := el.GetAttribute("innerHTML")
	if err != nil {
		return err
	}

	if *outfile != "" {
		if err := ioutil.WriteFile(*outfile, []byte(src), 0644); err != nil {
			return err
		}
		log.Printf("wrote to %s", *outfile)
	} else {
		fmt.Println(src)
	}

	return nil
}

func realMain(ctx context.Context) error {
	if *noserver {
		log.Printf("skipping local server")
		u := uri()
		if _, err := request.Get(u, nil); err != nil {
			return errors.Errorf("could not contact local server: %s", u)
		}
	} else {
		go func() {
			check.Err(startLocalServerr(ctx))
		}()
	}

	if err := waitForRender(); err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()
	flags.RequireString(dir, "dir")
	flags.RequireString(selector, "selector")
	check.Err(realMain(context.Background()))
}

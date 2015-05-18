package collect

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"bosun.org/opentsdb"
	"bosun.org/slog"
)

func queuer() {
	for dp := range tchan {
		qlock.Lock()
		for {
			if len(queue) > MaxQueueLen {
				slock.Lock()
				dropped++
				slock.Unlock()
				break
			}

			m, err := json.Marshal(dp)
			if err != nil {
				slog.Error(err)
			} else {
				convertedValue := 0.0
				fval, floatOK := dp.Value.(float64)
				if floatOK {
					convertedValue = fval
				}
				ival, intOK := dp.Value.(int)
				if intOK {
					convertedValue = float64(ival)
				}
				// log.Printf("\n\n======\n\nConverted to %v\n\n=====\n\n", convertedValue)

				// log.Printf("\n\tValue: %v\n\tToFloat? %v\n\tToInt? %v\n\tQueued %v\n\tas: %v\n", convertedValue, floatOK, intOK, dp, string(m))
				queue = append(queue, m)
				prtgQueue = append(prtgQueue, Result{Channel: dp.Metric, Value: convertedValue})
			}
			select {
			case dp = <-tchan:
				continue
			default:
			}
			break
		}
		qlock.Unlock()
	}
}

func send() {
	for {
		qlock.Lock()
		if i := len(queue); i > 0 {
			if i > BatchSize {
				i = BatchSize
			}
			// sending := queue[:i]
			queue = queue[i:]
			if Debug {
				slog.Infof("sending: %d, remaining: %d", i, len(queue))
			}

			prtgSending := prtgQueue[:i]
			prtgQueue = prtgQueue[i:]

			qlock.Unlock()
			// sendBatch(sending)
			sendPRTG(prtgSending)
		} else {
			qlock.Unlock()
			time.Sleep(time.Second)
		}
	}
}

func sendPRTG(batch []Result) {
	for _, result := range batch {
		add(result.Channel, result.Value)
	}
}

func pushToPRTGServer() {
	log.Printf("pushToPRTGServer")
	prtg := PRTG{Results: make([]Result, 0)}
	for key, value := range prtgResults {
		prtg.Results = append(prtg.Results, Result{key, value})
	}
	x, err := xml.MarshalIndent(prtg, "", "  ")
	if err != nil {
		slog.Error(err)
		return
	}
	xx := xml.Header + string(x)
	log.Printf("Sending %v to PRTG\n\n%v\n\n", len(prtg.Results), xx)

	prtgURL = "http://prtg.mayo.edu:5050/collector.dewey-integration.mayo.edu"
	req, err := http.NewRequest("POST", prtgURL, bytes.NewBuffer([]byte(xx)))
	if err != nil {
		slog.Error(err)
		return
	}
	req.Header.Set("Content-Type", "application/xml")
	// now := time.Now()
	resp, err := client.Do(req)
	// d := time.Since(now).Nanoseconds() / 1e6
	if err == nil {
		defer resp.Body.Close()
	}
	// _, _ = ioutil.ReadAll(resp.Body)
	// log.Printf("Got response: %v\n\n", string(body))
	// Add("collect.post.total_duration", Tags, d)
	// Add("collect.post.count", Tags, 1)
	// recordSent(len(prtg.Results))
	log.Printf("finished pushToPRTGServer")

}

func sendBatch(batch []json.RawMessage) {
	if Print {
		for _, d := range batch {
			slog.Info(string(d))
		}
		recordSent(len(batch))
		return
	}
	var buf bytes.Buffer
	g := gzip.NewWriter(&buf)
	if err := json.NewEncoder(g).Encode(batch); err != nil {
		slog.Error(err)
		return
	}
	if err := g.Close(); err != nil {
		slog.Error(err)
		return
	}
	req, err := http.NewRequest("POST", tsdbURL, &buf)
	if err != nil {
		slog.Error(err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	now := time.Now()
	resp, err := client.Do(req)
	d := time.Since(now).Nanoseconds() / 1e6
	if err == nil {
		defer resp.Body.Close()
	}
	Add("collect.post.total_duration", Tags, d)
	Add("collect.post.count", Tags, 1)
	// Some problem with connecting to the server; retry later.
	if err != nil || resp.StatusCode != http.StatusNoContent {
		if err != nil {
			Add("collect.post.error", Tags, 1)
			slog.Error(err)
		} else if resp.StatusCode != http.StatusNoContent {
			Add("collect.post.bad_status", Tags, 1)
			slog.Errorln(resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				slog.Error(err)
			}
			if len(body) > 0 {
				slog.Error(string(body))
			}
		}
		restored := 0
		for _, msg := range batch {
			var dp opentsdb.DataPoint
			if err := json.Unmarshal(msg, &dp); err != nil {
				slog.Error(err)
				continue
			}
			restored++
			tchan <- &dp
		}
		d := time.Second * 5
		Add("collect.post.restore", Tags, int64(restored))
		slog.Infof("restored %d, sleeping %s", restored, d)
		time.Sleep(d)
		return
	}
	recordSent(len(batch))
}

func recordSent(num int) {
	if Debug {
		slog.Infoln("sent", num)
	}
	slock.Lock()
	sent += int64(num)
	slock.Unlock()
}

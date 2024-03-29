package collectors

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"bosun.org/metadata"
	"bosun.org/opentsdb"
	"bosun.org/util"
)

var collectors []Collector

type Collector interface {
	Run(chan<- *opentsdb.DataPoint)
	Name() string
	Init()
}

const (
	osCPU          = "os.cpu"
	osCPUClock     = "os.cpu.clock"
	osDiskFree     = "os.disk.fs.space_free"
	osDiskPctFree  = "os.disk.fs.percent_free"
	osDiskTotal    = "os.disk.fs.space_total"
	osDiskUsed     = "os.disk.fs.space_used"
	osMemFree      = "os.mem.free"
	osMemPctFree   = "os.mem.percent_free"
	osMemTotal     = "os.mem.total"
	osMemUsed      = "os.mem.used"
	osNetBroadcast = "os.net.packets_broadcast"
	osNetBytes     = "os.net.bytes"
	osNetDropped   = "os.net.dropped"
	osNetErrors    = "os.net.errs"
	osNetMulticast = "os.net.packets_multicast"
	osNetPackets   = "os.net.packets"
	osNetUnicast   = "os.net.packets_unicast"
	osSystemUptime = "os.system.uptime"
)

const (
	osCPUClockDesc     = "The current speed of the processor in MHz."
	osDiskFreeDesc     = "The space_free property indicates in bytes how much free space is available on the disk."
	osDiskPctFreeDesc  = "The percent_free property indicates what percentage of the disk is available."
	osDiskTotalDesc    = "The space_total property indicates in bytes how much total space is on the disk."
	osDiskUsedDesc     = "The space_used property indicates in bytes how much space is used on the disk."
	osMemFreeDesc      = "Number, in bytes, of physical memory currently unused and available."
	osMemPctFreeDesc   = "The percent of free memory. In Linux free memory includes memory used by buffers and cache."
	osMemUsedDesc      = "The amount of used memory. In Linux this excludes memory used by buffers and cache."
	osNetBytesDesc     = "The rate at which bytes are sent or received over each network adapter."
	osNetDroppedDesc   = "The number of packets that were chosen to be discarded even though no errors had been detected to prevent transmission."
	osNetErrorsDesc    = "The number of packets that could not be transmitted because of errors."
	osNetPacketsDesc   = "The rate at which packets are sent or received on the network interface."
	osSystemUptimeDesc = "Seconds since last reboot."
)

var (
	// DefaultFreq is the duration between collection intervals if none is
	// specified.
	DefaultFreq = time.Second * 15

	timestamp = time.Now().Unix()
	tlock     sync.Mutex
	AddTags   opentsdb.TagSet

	AddProcessDotNetConfig = func(line string) error {
		return fmt.Errorf("process_dotnet watching not implemented on this platform")
	}
	WatchProcessesDotNet = func() {}
)

var (
	KeepAliveCommunity = "public"
)

func init() {
	go func() {
		for t := range time.Tick(time.Second) {
			tlock.Lock()
			timestamp = t.Unix()
			tlock.Unlock()
		}
	}()
}

func now() (t int64) {
	tlock.Lock()
	t = timestamp
	tlock.Unlock()
	return
}

// Search returns all collectors matching the pattern s, and exclude those matching e.
func Search(s string, e string) []Collector {
	var r []Collector
	for _, c := range collectors {
		matches := false
		excluded := false
		for _, p := range strings.Split(s, ",") {
			if strings.Contains(c.Name(), p) {
				matches = true
				break
			}
		}

		for _, p := range strings.Split(e, ",") {
			if p != "" && strings.Contains(c.Name(), p) {
				excluded = true
				break
			}
		}
		if matches && !excluded {
			r = append(r, c)
		}
	}
	return r
}

// Run runs specified collectors. Use nil for all collectors.
func Run(cs []Collector) chan *opentsdb.DataPoint {
	if cs == nil {
		cs = collectors
	}
	ch := make(chan *opentsdb.DataPoint)
	for _, c := range cs {
		go c.Run(ch)
	}
	return ch
}

type MetricMeta struct {
	Metric   string
	TagSet   opentsdb.TagSet
	RateType metadata.RateType
	Unit     metadata.Unit
	Desc     string
}

// AddTS is the same as Add but lets you specify the timestamp
func AddTS(md *opentsdb.MultiDataPoint, name string, ts int64, value interface{}, t opentsdb.TagSet, rate metadata.RateType, unit metadata.Unit, desc string) {
	tags := t.Copy()
	if rate != metadata.Unknown {
		metadata.AddMeta(name, nil, "rate", rate, false)
	}
	if unit != metadata.None {
		metadata.AddMeta(name, nil, "unit", unit, false)
	}
	if desc != "" {
		metadata.AddMeta(name, tags, "desc", desc, false)
	}
	if host, present := tags["host"]; !present {
		tags["host"] = util.Hostname
	} else if host == "" {
		delete(tags, "host")
	}
	tags = AddTags.Copy().Merge(tags)
	d := opentsdb.DataPoint{
		Metric:    name,
		Timestamp: ts,
		Value:     value,
		Tags:      tags,
	}
	*md = append(*md, &d)
}

// Add appends a new data point with given metric name, value, and tags. Tags
// may be nil. If tags is nil or does not contain a host key, it will be
// automatically added. If the value of the host key is the empty string, it
// will be removed (use this to prevent the normal auto-adding of the host tag).
func Add(md *opentsdb.MultiDataPoint, name string, value interface{}, t opentsdb.TagSet, rate metadata.RateType, unit metadata.Unit, desc string) {
	AddTS(md, name, now(), value, t, rate, unit, desc)
}

func readLine(fname string, line func(string) error) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := line(scanner.Text()); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// IsDigit returns true if s consists of decimal digits.
func IsDigit(s string) bool {
	r := strings.NewReader(s)
	for {
		ch, _, err := r.ReadRune()
		if ch == 0 || err != nil {
			break
		} else if ch == utf8.RuneError {
			return false
		} else if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

// IsAlNum returns true if s is alphanumeric.
func IsAlNum(s string) bool {
	r := strings.NewReader(s)
	for {
		ch, _, err := r.ReadRune()
		if ch == 0 || err != nil {
			break
		} else if ch == utf8.RuneError {
			return false
		} else if !unicode.IsDigit(ch) && !unicode.IsLetter(ch) {
			return false
		}
	}
	return true
}

func TSys100NStoEpoch(nsec uint64) int64 {
	nsec -= 116444736000000000
	seconds := nsec / 1e7
	return int64(seconds)
}

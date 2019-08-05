// m3u_tst
package main

// #EXT-X-ENDLIST у live этого в конце нет
// #EXT-X-MEDIA-SEQUENCE - порядковый номер первого uri в списке воспроизвед., начин с 1 для live

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"m3u8"
	"strconv"
	"strings"

	"bytes"
	//"crypto/tls"
	"h12.io/socks"

	//"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"
)

type SVHttpIO struct {
	proxyurl string
	client   *http.Client
}

func (t *SVHttpIO) SetProxy(proxyurl string) {
	proxy := proxyurl
	url_proxy, err := url.Parse(proxy)
	if err != nil {
		fmt.Println("Proxy url:", err.Error())
	}

	var tr http.Transport
	if (url_proxy.Scheme == "http") || (url_proxy.Scheme == "https") {
		tr = http.Transport{}
		tr.Proxy = http.ProxyURL(url_proxy)
		//tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //set ssl
		fmt.Println("http proxy")
	} else {
		dialSocksProxy := socks.Dial(proxy)
		tr = http.Transport{Dial: dialSocksProxy}
		fmt.Println("socks proxy")
	}
	t.client = &http.Client{}
	t.client.Transport = &tr
}

func (t *SVHttpIO) Init(proxyurl string) {
	if proxyurl == "" {
		t.client = http.DefaultClient
		return
	}

	t.SetProxy(proxyurl)

	fmt.Printf("set proxy=%s\n", t.proxyurl)

}

func (t *SVHttpIO) httpFile(_url string) (_data []byte, merr int) {
	ret := 20
	for true { //for n := 0; n < 20; n++ {
		res, err := t.client.Get(_url)
		if err != nil {
			//log.Fatal(err)
			fmt.Printf("httpFile eror Get: %s\n", err.Error())
			ret = 1
			//return _data, 1
		} else {
			if res.StatusCode != 200 {
				fmt.Printf("httpFile eror Get StatusCode=%d\n", res.StatusCode)
				//return _data, 2
				ret = 2
			} else {
				_data, err = ioutil.ReadAll(res.Body)
				res.Body.Close()
				if err != nil {
					//log.Fatal(err)
					fmt.Printf("httpFile eror=%d\n", 2)
					//return _data, 5
					ret = 5
				} else {
					ret = 0
					break
				}
			}
		}
		time.Sleep(1000000 * 2000) //4 sec
	}
	//fmt.Printf("%s", robots)
	return _data, ret
}

//----
type SVSegDownl struct {
	segs  *SVVideoSegments
	wfile *os.File
	rio   *SVHttpIO
}

func (t *SVSegDownl) Open(wfname string, rio *SVHttpIO, segs *SVVideoSegments) {
	t.wfile, _ = os.Create(wfname)
	t.rio = rio
	t.segs = segs
}

//download 1 segment
func (t *SVSegDownl) Down() (merr int) {
	ds := t.segs.lastdlid + 1
	idx := ds - 1
	//var pseg *SVVideoSegment
	if idx >= uint64(len(t.segs.segs)) {
		fmt.Printf("No more seg\n")
		return 1
	}
	pseg := &t.segs.segs[idx]

	if pseg.furl == "" {
		//fmt.Printf("Null segment not download id=%d\n", pseg.idseg)
		t.segs.lastdlid = ds
		return 0
	}

	if pseg.idseg != ds {
		fmt.Printf("Error id: idx=%d idsec=%d ds=%d\n", idx, pseg.idseg, ds)
		return 2
	}

	buf, merr := t.rio.httpFile(pseg.furl)
	if merr == 0 {

		_, _ = t.wfile.Write(buf)

		//fmt.Printf("Dl seg=%d\n", pseg.idseg)
		t.segs.lastdlid = ds

		return 0
	} else {
		fmt.Printf("Error Dl seg=%d\n", pseg.idseg)
		t.segs.lastdlid = ds
		return 0
	}
	return 10

}

//----
type SVVideoSegment struct {
	idseg  uint64
	furl   string
	lensec float64
}

//----
type SVVideoSegments struct {
	segs     []SVVideoSegment
	lastid   uint64
	lastdlid uint64 //псоледн. скач. сегмент
}

func (t *SVVideoSegments) Add(idseg uint64, furl string, lensec float64) (merr int) {
	merr = 0
	if idseg > t.lastid {
		var seg SVVideoSegment
		//in this case add null segments
		if idseg-t.lastid > 1 {
			//n := idseg - t.lastid
			for n := t.lastdlid + 1; n < idseg; n++ {
				seg.furl = ""
				seg.idseg = n
				seg.lensec = 0
				t.segs = append(t.segs, seg)
				//fmt.Printf("Add null segment id=%d\n", n)
				t.lastid = n
			}
		}

		seg.furl = furl
		seg.idseg = idseg
		seg.lensec = lensec
		t.segs = append(t.segs, seg)
		//fmt.Printf("Add segment id=%d\n", seg.idseg)

		//DownlSeg(idseg,furl)
		t.lastid = idseg

		merr = 1
	}
	return merr
}

func (t *SVVideoSegments) ToFile(fname string) {
	wfile, _ := os.Create(fname)

	//b, _ := json.Marshal(t.svpl.segs.segs)
	//wfile.Write(b)

	for _, ms := range t.segs {
		fmt.Fprintf(wfile, "%s;%d;%f\n", ms.furl, ms.idseg, ms.lensec)
	}

	wfile.Close()
}
func (t *SVVideoSegments) FromFile(fname string) {
	wfile, _ := os.Open(fname)
	///b, _ := json.Marshal(t.svpl.segs.segs)

	//b, _ := ioutil.ReadAll(wfile)

	var furl string
	var idseg uint64
	var lensec float64
	/*
		for true {
			_, err := fmt.Fscanln(wfile, "%s %d %f\n", &furl, &idseg, &lensec)
			if err != nil {
				fmt.Println(err.Error())
				break
			}
			t.svpl.segs.Add(idseg, furl, lensec)
			fmt.Println("ld", idseg, furl, lensec, "\n")
		}
	*/

	sc := bufio.NewScanner(wfile)
	for sc.Scan() {
		s := sc.Text() // GET the line string
		ss := strings.Split(s, ";")
		furl = ss[0]
		idseg, _ = strconv.ParseUint(ss[1], 10, 64)
		lensec, _ = strconv.ParseFloat(ss[2], 64)
		t.Add(idseg, furl, lensec)
		fmt.Println("ld", idseg, furl, lensec, "\n")

		//fmt.Println(ss)

	}

	wfile.Close()
}

//----

type SVPlayList struct {
	//m3u8
	//fullurl string
	fdata   []byte
	baseurl string

	segs SVVideoSegments

	curtargdur float64
}

//func (t *SVPlayList) Init(_url string) {
//	t.fullurl = _url
//}

//download and parse m3u
func (t *SVPlayList) Parse(_url string, io *SVHttpIO) (merr int) {
	merr = 0
	//f, err := os.Open(t.fullurl)
	//if err != nil {
	//	fmt.Println(err)
	//	merr = 10
	//}

	//u, _ := url.Parse(_url)
	//u.

	//p := m3u8.NewMasterPlaylist()
	var dt []byte
	dt, merr = io.httpFile(_url)
	if merr != 0 {
		return 1
	}
	pll, plltype, err := m3u8.Decode(*bytes.NewBuffer(dt), false)
	if err != nil {
		fmt.Printf("Error m3u decode: ")
		fmt.Println(err)
		merr = 2
		return 2
	}

	//fmt.Printf("Playlist object:\n %+v \n type=%v\n", pll, plltype)

	switch plltype {
	case m3u8.MEDIA:
		medpl := pll.(*m3u8.MediaPlaylist)

		//var ms *m3u8.MediaSegment
		//medpl.TargetDuration - макимальная продолжительность сегмента в секундах
		fmt.Printf("segno=%d targetdur=%f winsize=%d\n", medpl.SeqNo, medpl.TargetDuration, medpl.WinSize())
		fmt.Printf("Segments in playlist count=%d\n", medpl.Count())

		t.curtargdur = medpl.TargetDuration

		for _, ms := range medpl.Segments {
			if ms == nil {
				break
			}
			//fmt.Printf("%+v\n", ms)
			furl, _ := url.Parse(_url)
			surl, _ := url.Parse(ms.URI)
			var segurl *url.URL
			if surl.IsAbs() == false {
				segurl = furl.ResolveReference(surl)
			} else {
				segurl = surl
			}

			//fmt.Printf("segid=%d title=%s dur=%f, uri=%s\n", ms.SeqId, ms.Title, ms.Duration, ms.URI)
			//fmt.Printf("furl=%s\n", segurl.String())

			t.segs.Add(ms.SeqId, segurl.String(), ms.Duration)
		}

		//medpl.
	case m3u8.MASTER:
		maspl := pll.(*m3u8.MasterPlaylist)
		fmt.Printf("%+v\n", maspl)
	}

	return merr
}

//---------------
type SVM3UManag struct {
	svpl SVPlayList
	io   *SVHttpIO
	dl   SVSegDownl

	csig chan os.Signal

	wrflag       int
	savesegsflag int
	frseg        uint64
}

func (t *SVM3UManag) DownAll() {
	for t.dl.Down() == 0 {
	}
}

/*
func (t *SVM3UManag) SaveSegments(fname string) {
	wfile, _ := os.Create(fname)

	//b, _ := json.Marshal(t.svpl.segs.segs)
	//wfile.Write(b)

	for _, ms := range t.svpl.segs.segs {
		fmt.Fprintf(wfile, "%s;%d;%f\n", ms.furl, ms.idseg, ms.lensec)
	}

	wfile.Close()
}
*/
/*
func (t *SVM3UManag) LoadSegments(fname string) {
	wfile, _ := os.Open(fname)
	///b, _ := json.Marshal(t.svpl.segs.segs)

	//b, _ := ioutil.ReadAll(wfile)

	var furl string
	var idseg uint64
	var lensec float64

	sc := bufio.NewScanner(wfile)
	for sc.Scan() {
		s := sc.Text() // GET the line string
		ss := strings.Split(s, ";")
		furl = ss[0]
		idseg, _ = strconv.ParseUint(ss[1], 10, 64)
		lensec, _ = strconv.ParseFloat(ss[2], 64)
		t.svpl.segs.Add(idseg, furl, lensec)
		fmt.Println("ld", idseg, furl, lensec, "\n")

		//fmt.Println(ss)

	}

	wfile.Close()
}
*/

func (t *SVM3UManag) IsCntrlC() bool {
	if len(t.csig) > 0 {
		return true
	}
	return false

}

//set number of segment from start download
func (t *SVM3UManag) SetDlFrom(segid uint64) {
	//t.dl.segs.lastdlid = segid - 1
	t.frseg = segid
}

//reflag==0 - live 1-record
func (t *SVM3UManag) Run(url string, wrfname string, wrflag int, recflag int, fromsegfileflag int) {
	t.csig = make(chan os.Signal, 1)
	signal.Notify(t.csig, os.Interrupt)

	if fromsegfileflag == 1 {
		recflag = 1
	}

	t.wrflag = wrflag

	t.io = new(SVHttpIO)
	t.io.Init("")

	if t.wrflag == 1 {
		t.dl.Open(wrfname, t.io, &t.svpl.segs)
	}

	if t.frseg > 0 {
		t.dl.segs.lastdlid = t.frseg
		fmt.Println("Dl from segment:", t.frseg)
	}

	var dlcnt = 0
	for true {

		if fromsegfileflag != 1 {
			t.svpl.Parse(url, t.io)
		} else {
			t.svpl.segs.FromFile("segs.dat")
			fmt.Println("Load segments from file \n")
		}

		fmt.Printf("Add segments count=%d\n", len(t.svpl.segs.segs))

		tmd := t.svpl.curtargdur
		tmdms := int64(t.svpl.curtargdur*1000) + 200 //+ 200 ms for best
		tm := time.Now()

		if t.wrflag == 1 {
			//t.DownAll()
			for t.dl.Down() == 0 {
				dlcnt++
				if dlcnt%10 == 0 {
					fmt.Printf("Dl seg cnt=%d\n", dlcnt)
				}
				if t.IsCntrlC() == true {
					break
				}
			}
		}

		//if recording, is over
		if recflag == 1 {
			break
		}

		tm2 := time.Now()
		tmr := tm2.Sub(tm)

		if tmr.Seconds() < tmd {
			t := tmdms - tmr.Nanoseconds()/1000000
			time.Sleep(time.Duration(t) * time.Millisecond)
			fmt.Printf("Wait %d ms\n", t)
		}

		if t.IsCntrlC() == true {
			break
		}

	}

	if t.wrflag == 1 {
		//t.dl.Close()
	}

	if t.savesegsflag == 1 {
		//t.SaveSegments("segs.dat")
		t.svpl.segs.ToFile("segs.dat")
	}

	fmt.Printf("Close OK\n")
}

//-----------------

func main() {
	fmt.Println("SVM3UDN")
	fmt.Println("SV Soft, nevidprogr@gmail.com\n\n")

	//io := new(SVHttpIO)
	//proxy := "https://86.57.181.122:48890"  //!!!work
	//proxy := "https://86.57.181.12:48890"
	//proxy = "socks5://104.237.227.198:54321"
	//io.Init("")
	//io.Init(proxy)

	//url := "https://live-208.zxz.su/LIVE/LIVE:320319/HLS/1562186560/e4rj1ZHBNZ_tNXbB_lJ6kQ/live/320319_1/chunklist.m3u8"

	//url := "https://live-304.zxz.su/LIVE/LIVE:320385/HLS/1562248540/QJHJsbocSCwxH5_XjRI3yA/live/320385_1/chunklist.m3u8"

	var url string
	var rec int
	var dlf int
	var frseg uint64
	var fromsegfile int
	var savesegfile int
	var proxy string
	flag.StringVar(&url, "url", "", "m3u url")
	flag.StringVar(&proxy, "proxy", "", "")
	flag.IntVar(&rec, "rec", 0, "")
	flag.IntVar(&dlf, "dl", 0, "")
	flag.Uint64Var(&frseg, "frseg", 0, "")
	flag.IntVar(&fromsegfile, "fromsegfile", 0, "")
	flag.IntVar(&savesegfile, "savesegfile", 0, "")
	flag.Parse()
	flag.PrintDefaults()
	fmt.Printf("url=%s rec=%d dl=%d frseg=%d fromsegfile=%d\n", url, rec, dlf, frseg, fromsegfile)
	//return

	io := new(SVHttpIO)
	io.Init(proxy)

	var man SVM3UManag

	//man.LoadSegments("segs.dat")
	//return

	if frseg > 0 {
		man.SetDlFrom(frseg)
	}
	if savesegfile == 1 {
		man.savesegsflag = 1
	}

	man.Run(url, "test.ts", dlf, rec, fromsegfile)

}

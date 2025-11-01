package main

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	//go:embed adbd
	adbdBytes []byte
)

const (
	defaultIP       = "192.168.100.1"
	defaultBytesCnt = 128
	targetPath      = "/mnt/userdata"
	targetFile      = targetPath + "/adbd"
)

type Executer struct {
	IP string
}

func (e *Executer) runCmd(cmd string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/reqproc/proc_post?goformId=ALK_EXC_SYSTEM_CMD&SYS_CMD=%s", e.IP, url.QueryEscape(cmd)))
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(bodyBytes))
}

func (e *Executer) runAT1(cmd string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/reqproc/proc_post?goformId=ALK_EXC_AT_CMD1&AT_CMD1=%s", e.IP, url.QueryEscape(cmd)))
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(bodyBytes))
}

func (e *Executer) runAT2(cmd string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/reqproc/proc_post?goformId=ALK_EXC_AT_CMD2&AT_CMD2=%s", e.IP, url.QueryEscape(cmd)))
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(bodyBytes))
}

func (e *Executer) Remount() {
	e.runCmd("mount -o remount,rw " + targetPath)
	e.runCmd("cp /bin/busybox " + targetFile)
}

func (e *Executer) Push() {
	e.runCmd("printf '' >" + targetFile)
	builder := &strings.Builder{}
	for i := range adbdBytes {
		fmt.Fprintf(builder, "\\x%02x", adbdBytes[i])
		if i%defaultBytesCnt == 0 {
			e.runCmd(fmt.Sprintf("printf '%s'>>"+targetFile, builder.String()))
			builder = &strings.Builder{}
			log.Printf("%%%.2f\n", float32(i)*float32(100.0)/float32(len(adbdBytes)))
		}
		if i == len(adbdBytes)-1 {
			e.runCmd(fmt.Sprintf("printf '%s'>>"+targetFile, builder.String()))
		}
	}
}

func (e *Executer) Enable() {
	e.runCmd("chmod 777 " + targetFile)
	//e.runCmd("/bin/sh -c " + targetFile + " &")
	e.runAT1("shell="+targetFile)
}

func start(ip string, onlyStart bool) {
	e := Executer{ip}
	if !onlyStart {
		e.Remount()
		e.Push()
	}
	go e.Enable()
}

func main() {
	var (
		ip        string
		onlystart bool
	)
	flag.StringVar(&ip, "ip", defaultIP, "后台地址")
	flag.BoolVar(&onlystart, "s", false, "只启动adbd服务而不推送")
	flag.Parse()
	if ip == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("使用地址：http://%s，按任意键继续", ip)
	bufio.NewReader(os.Stdin).ReadString('\n')
	start(ip, onlystart)
	log.Println("执行结束，按任意键退出")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

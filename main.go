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
	"time"
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
	log.Println("runCMD:", cmd)
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

func (e *Executer) setConfig(key, value string) {
	e.runCmd(fmt.Sprintf("nv set %s=%s", key, value))
}

func (e *Executer) setConfigAndSave(key, value string) {
	e.setConfig(key, value)
	e.saveConfig()
}

func (e *Executer) saveConfig() {
	e.runCmd("nv save")
}

func (e *Executer) runAT1(cmd string) {
	log.Println("run AT:",cmd)
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
	log.Println("[+] Start push adbd...")
	e.runCmd("printf '' >" + targetFile)
	builder := &strings.Builder{}
	// 如果下发策略被ban，执行下面逻辑
	// e.runCmd(`sleep 20; ifconfig usblan0 up;`)
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

func (e *Executer) DevMode(mode int) {
	log.Println("[+] Set dev mode")
	resp, err := http.Get(fmt.Sprintf("http://%s/reqproc/proc_post?goformId=SET_DEVICE_MODE&debug_enable=%d", e.IP, mode))
	if err != nil {
		log.Fatal(err)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(bodyBytes))
}

func (e *Executer) Enable() {
	e.DevMode(1)
	e.runCmd("chmod 777 " + targetFile)
	e.runCmd("/bin/sh -c " + targetFile + " &")
	//e.runAT1("shell=" + targetFile)
}

func start(ip string, onlyStart bool) {
	e := Executer{ip}
	if !onlyStart {
		e.Remount()
		e.Push()
	}
	go e.Enable()
}

func switchCard(ip string, Id int) {
	e := Executer{ip}
	e.setConfigAndSave("slot_in", "0")
	e.runAT1("CFUN=5")
	time.Sleep(time.Second)
	switch Id {
	case 1:
		e.setConfigAndSave("alk_sim_current", "1")
		e.runCmd("echo 0 > /sys/class/gpio/gpio127/value")
		e.runAT1("ZCARDSWITCH=0,0")
		// e.setConfig("wan_apn", "")
	case 2:
		e.setConfig("alk_sim_current", "2")
		e.runCmd("echo 0 > /sys/class/gpio/gpio127/value")
		e.runAT1("ZCARDSWITCH=3,0")
	case 3:
		e.setConfig("alk_sim_current", "3")
		e.runCmd("echo 1 > /sys/class/gpio/gpio127/value")
		e.runAT1("ZCARDSWITCH=3,0")
	default:
		e.setConfig("alk_sim_current", "1")
		e.runCmd("echo 0 > /sys/class/gpio/gpio127/value")
		e.runAT1("ZCARDSWITCH=0,0")
		log.Println("unsupport id:", Id)
	}
	e.saveConfig()
	e.setConfigAndSave("alk_sim_flag","0")
	e.setConfigAndSave("mmi_close_freq","1")
	e.runAT1("CFUN=0")
	time.Sleep(3 * time.Second)
	e.runAT1("CFUN=1")
	e.setConfig("switch_card_complete", "1")
}

func main() {
	var (
		ip           string
		onlystart    bool
		switchCardId int
	)
	flag.StringVar(&ip, "ip", defaultIP, "后台地址")
	flag.BoolVar(&onlystart, "s", false, "只启动adbd服务而不推送")
	flag.IntVar(&switchCardId, "switch", -1, "切换sim卡")

	flag.Parse()
	if ip == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("使用地址：http://%s，按任意键继续", ip)
	bufio.NewReader(os.Stdin).ReadString('\n')
	if switchCardId != -1 {
		switchCard(ip, switchCardId)
	} else {
		start(ip, onlystart)
	}

	log.Println("执行结束，按任意键退出")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

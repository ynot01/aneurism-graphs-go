package main

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rumblefrog/go-a2s"
)

const MASTER_URL = "https://content.aneurismiv.com/masterlist"

type dataPoint struct {
	time_millis int64
	rot         uint8
	players     uint8
	rotted      bool
}

type Server struct {
	name string
	data []*dataPoint
}

// udpClients["0.0.0.0:7777"] = *a2s.Client
var udpClients map[string]*a2s.Client = make(map[string]*a2s.Client)

// udpClients["0.0.0.0:7777"] = Server{}
var registeredServers map[string]*Server = make(map[string]*Server)

// Loops through all official servers and queries them one by one
func main() {
	fmt.Printf("[%v] Started aneurism-graphs-go.\n", time.Now().Format(time.RFC850))
	{
		_, nodeCheckErr := exec.LookPath("node")
		if nodeCheckErr != nil {
			fmt.Printf("[%v] nodeCheckErr: %v\n", time.Now().Format(time.RFC850), nodeCheckErr)
			return
		}
	}
	fmt.Printf("[%v] Found Node.\n", time.Now().Format(time.RFC850))
	{
		_, npmCheckErr := exec.LookPath("npm")
		if npmCheckErr != nil {
			fmt.Printf("[%v] npmCheckErr: %v\n", time.Now().Format(time.RFC850), npmCheckErr)
			return
		}
	}
	fmt.Printf("[%v] Found NPM.\n", time.Now().Format(time.RFC850))
	{
		installCheck := exec.Command("npm", "install")
		installCheck.Dir = "./aneurism-graphs/"
		errBuilder := new(strings.Builder)
		installCheck.Stderr = errBuilder
		installCheckErr := installCheck.Run()
		if installCheckErr != nil {
			fmt.Printf("[%v] installCheckErr: %v\n", time.Now().Format(time.RFC850), installCheckErr)
			return
		}
		builtInstallStdErr := errBuilder.String()
		if builtInstallStdErr != "" {
			fmt.Printf("[%v] installCheck Stderr is not empty: %v\n", time.Now().Format(time.RFC850), builtInstallStdErr)
			return
		}
	}
	fmt.Printf("[%v] Installed npm packages.\n", time.Now().Format(time.RFC850))
	{
		surgeCheck := exec.Command("surge", "whoami")
		output, surgeCheckErr := surgeCheck.CombinedOutput()
		if surgeCheckErr != nil {
			fmt.Printf("[%v] surgeCheckErr: %v\n", time.Now().Format(time.RFC850), surgeCheckErr)
			return
		}
		if !strings.Contains(string(output), "Student") {
			fmt.Printf("[%v] Surge output does not contain \"Student\": %v\n", time.Now().Format(time.RFC850), string(output))
			return
		}
	}
	fmt.Printf("[%v] Logged into surge.sh.\n", time.Now().Format(time.RFC850))
	main_loop()
	for range time.Tick(time.Second * 300) { // Wait a healthy 5 minutes
		main_loop()
	}
}

func main_loop() {
	official_servers := get_masterlist()
	for official := range official_servers {
		ipAddr := official_servers[official]
		if strings.TrimSpace(ipAddr) == "" {
			continue
		}
		findComment := strings.Index(ipAddr, "//") // Strip comments from masterlist
		if findComment != -1 {
			ipAddr = strings.TrimSpace(ipAddr[:findComment])
		}
		dictKey := ipAddr
		findPort := strings.Index(ipAddr, ":") // Isolate :port (and ignore entries without a port)
		var ipPort string
		if findPort == -1 {
			ipPort = "7777"
		} else {
			ipPort = strings.TrimSpace(ipAddr[findPort:])[1:]
		}
		portInt, atoiErr := strconv.Atoi(ipPort)
		if atoiErr != nil {
			fmt.Printf("[%v] atoiErr: %v\n", time.Now().Format(time.RFC850), atoiErr)
			continue
		}
		if findPort == -1 {
			ipAddr = fmt.Sprintf("%v:%v", ipAddr, portInt+1)
		} else {
			ipAddr = fmt.Sprintf("%v:%v", ipAddr[:findPort], portInt+1) // Add 1 to server port to get the a2s query port
		}
		client, weHaveClient := udpClients[dictKey]
		if !weHaveClient {
			newClient, newClientErr := a2s.NewClient(
				ipAddr,
				a2s.SetAppID(2773280),
			)
			if newClientErr != nil {
				fmt.Printf("[%v] newClientErr: %v\n", time.Now().Format(time.RFC850), newClientErr)
				continue
			}
			client = newClient
			udpClients[dictKey] = newClient
		}
		info, infoErr := client.QueryInfo()
		if infoErr != nil {
			// Don't print on server connection errors- a few of them are down a lot
			// fmt.Printf("%v \"fail\"\n", ipAddr)
			continue
		} else {
			// fmt.Printf("%v \"success\"\n", ipAddr)
		}
		newDataPoint := new(dataPoint)
		newDataPoint.time_millis = time.Now().UnixMilli()
		newDataPoint.players = info.Players
		newDataPoint.rot = get_rot_from_keywords(info.ExtendedServerInfo.Keywords)
		newDataPoint.rotted = false
		// Servers are stored by server port, not query port!
		oldServer, serverIsRegistered := registeredServers[dictKey]
		if serverIsRegistered {
			if oldServer.name != fmt.Sprintf("%v %v - %v", get_region_from_keywords(info.ExtendedServerInfo.Keywords), info.Name, dictKey) {
				// server rotted
				newDataPoint.rotted = true
			}
			oldServer.name = fmt.Sprintf("%v %v - %v", get_region_from_keywords(info.ExtendedServerInfo.Keywords), info.Name, dictKey)
			oldServer.data = append(oldServer.data, newDataPoint)
			if (len(oldServer.data) > 288) || (oldServer.data[0].time_millis < (time.Now().UnixMilli() - 86400000)) {
				_, oldServer.data = oldServer.data, oldServer.data[1:]
			}
			registeredServers[dictKey] = oldServer
		} else {
			fmt.Printf("[%v] New server added to register: %v - %v\n", time.Now().Format(time.RFC850), dictKey, info.Name)
			myServer := new(Server)
			myServer.name = fmt.Sprintf("%v %v - %v", get_region_from_keywords(info.ExtendedServerInfo.Keywords), info.Name, dictKey)
			myServer.data = []*dataPoint{newDataPoint}
			registeredServers[dictKey] = myServer
		}
	}
	{
		dataTs := constructDataTs()   // git submodule update --recursive --remote
		dataWriteErr := os.WriteFile( // git submodule foreach --recursive git reset --hard
			"./aneurism-graphs/src/data.ts",
			dataTs,
			2,
		)
		if dataWriteErr != nil {
			fmt.Printf("[%v] dataWriteErr: %v\n", time.Now().Format(time.RFC850), dataWriteErr)
			return
		}
	}
	{
		webBuildCheck := exec.Command("npm", "run", "build")
		webBuildCheck.Dir = "./aneurism-graphs/"
		errBuilder := new(strings.Builder)
		webBuildCheck.Stderr = errBuilder
		webBuildCheckErr := webBuildCheck.Run()
		if webBuildCheckErr != nil {
			fmt.Printf("[%v] webBuildCheckErr: %v\n", time.Now().Format(time.RFC850), webBuildCheckErr)
			return
		}
	}
	{
		webPushCheck := exec.Command("surge", "./aneurism-graphs/site/", "a4tracker.surge.sh")
		errBuilder := new(strings.Builder)
		webPushCheck.Stderr = errBuilder
		webPushCheckErr := webPushCheck.Run()
		if webPushCheckErr != nil {
			fmt.Printf("[%v] webPushCheckErr: %v\n", time.Now().Format(time.RFC850), webPushCheckErr)
			return
		}
	}
}

// https://stackoverflow.com/a/66751055/15363640
// https://creativecommons.org/licenses/by-sa/4.0/
func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// Download updated official server list and split by newline
func get_masterlist() []string {
	resp, masterErr := http.Get(MASTER_URL)
	if masterErr != nil {
		fmt.Printf("[%v] masterErr: %v\n", time.Now().Format(time.RFC850), masterErr)
		return []string{}
	}
	resBody, ioErr := io.ReadAll(resp.Body)
	if ioErr != nil {
		fmt.Printf("[%v] ioErr: %v\n", time.Now().Format(time.RFC850), ioErr)
		return []string{}
	}
	return removeDuplicateStr(strings.Split(string(resBody), "\n"))
}

// String operations to isolate region
// region:us,uptime:0,protected:1,rot:0.06 = us
// region:au,uptime:30,protected:1,rot:0.05 = au
// region:cn,uptime:30,protected:1,rot:0.06 = cn
func get_region_from_keywords(keywords string) string {
	stringRegion := "Unknown Region"
	keywordStrings := strings.Split(strings.TrimSpace(keywords), ",")
	for n := range keywordStrings {
		keyAndValue := strings.Split(keywordStrings[n], ":")
		key := keyAndValue[0]
		value := keyAndValue[1]
		if key == "region" {
			stringRegion = value
		}
	}
	switch strings.ToUpper(stringRegion) {
	case "US":
		return "🇺🇸"
	case "AU":
		return "🇦🇺"
	case "CN":
		return "🇨🇳"
	case "RU":
		return "🇷🇺"
	case "EUROPEANUNION":
		return "🇪🇺"
	}
	return stringRegion
}

// String operations to isolate rot
// region:us,uptime:0,protected:1,rot:0.06 = 6
// region:au,uptime:30,protected:1,rot:0.05 = 5
// region:cn,uptime:30,protected:1,rot:0.06 = 6
func get_rot_from_keywords(keywords string) uint8 {
	stringRegion := uint8(0)
	keywordStrings := strings.Split(strings.TrimSpace(keywords), ",")
	for n := range keywordStrings {
		keyAndValue := strings.Split(keywordStrings[n], ":")
		key := keyAndValue[0]
		value := keyAndValue[1]
		if key == "rot" {
			rotRawFloat, converr := strconv.ParseFloat(value, 64)
			if converr != nil {
				fmt.Printf("[%v] converr: %v\n", time.Now().Format(time.RFC850), converr)
				continue
			}
			if rotRawFloat > 1.0 {
				fmt.Printf("[%v] rotRawFloat is over 1.0: %v\n", time.Now().Format(time.RFC850), keywords)
				continue
			}
			stringRegion = uint8(math.Floor(rotRawFloat * 100.0))
		}
	}
	return stringRegion
}

type Alphabetic []*Server

func (list Alphabetic) Len() int { return len(list) }

func (list Alphabetic) Swap(i, j int) { list[i], list[j] = list[j], list[i] }

func (list Alphabetic) Less(i, j int) bool {
	var si string = list[i].name
	var sj string = list[j].name
	var si_lower = strings.ToLower(si)
	var sj_lower = strings.ToLower(sj)
	if si_lower == sj_lower {
		return si > sj
	}
	return si_lower > sj_lower
}

func constructDataTs() []byte {
	var outBytes []byte
	outBytes = fmt.Append(outBytes, `
import { ChartDataset } from 'chart.js/auto';

export const servers = new Map<string, ChartDataset[]>;
export const serverRots = new Map<string, number[]>;

`)
	lastUpdated := int64(0)
	var servers []*Server
	for _, v := range registeredServers {
		servers = append(servers, v)
	}
	sort.Sort(Alphabetic(servers))
	for _, v := range servers {
		outBytes = fmt.Appendf(outBytes, `serverRots.set('%v',[`, v.name)
		for _, data := range v.data {
			if data.rotted {
				outBytes = fmt.Appendf(outBytes, "%v,\n", data.time_millis)
			}
		}
		outBytes = fmt.Appendf(outBytes, `])
		servers.set('%v',[{
            label: 'Players',
            data: [`, v.name)
		for _, data := range v.data {
			outBytes = fmt.Appendf(outBytes, "{x: %v, y: %v},\n", data.time_millis, data.players)
			if data.time_millis > lastUpdated {
				lastUpdated = data.time_millis
			}
		}
		outBytes = fmt.Appendf(outBytes, `
			],
            fill: false,
            borderColor: '#0077aa',
            backgroundColor: '#04b4ff',
            tension: 0.2,
            pointRadius: 2
        },
        {
            label: 'Rot',
            data: [`)
		for _, data := range v.data {
			outBytes = fmt.Appendf(outBytes, "{x: %v, y: %v},\n", data.time_millis, data.rot)
		}
		outBytes = fmt.Append(outBytes, `],
            fill: false,
            borderColor: '#aa5b00',
            backgroundColor: '#ff8800',
            tension: 0.2,
            pointRadius: 2
        },
    ]
	)
`)
	}
	outBytes = fmt.Appendf(outBytes, "export const lastUpdated = %v;", lastUpdated)
	return outBytes
}

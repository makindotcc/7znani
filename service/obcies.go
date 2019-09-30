package service

import (
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

const serverAddress = "wss://server.6obcy.pl:%d/6eio/?EIO=3&transport=websocket"

type ObcyPool struct {
	obcyList []*Obcy
	mutex    *sync.RWMutex
}

func NewObcyPool() *ObcyPool {
	return &ObcyPool{
		obcyList: make([]*Obcy, 0),
		mutex:    &sync.RWMutex{},
	}
}

func (pool *ObcyPool) Receive() *Obcy {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	if len(pool.obcyList) == 0 {
		return nil
	}
	obcy := pool.obcyList[0]
	pool.obcyList = pool.obcyList[1:]

	return obcy
}

func (pool *ObcyPool) Put(obcy *Obcy) {
	if obcy.closed {
		return
	}
	pool.mutex.Lock()
	pool.obcyList = append(pool.obcyList, obcy)
	pool.mutex.Unlock()
}

type Obcy struct {
	client                       *websocket.Conn
	ceid                         int
	ckey                         string
	messageId                    int
	packetHandler                PacketHandler
	messageListener              func(message string)
	strangerConnectedListener    func()
	strangerDisconnectedListener func()
	strangerTypingStatusListener func(status bool)
	closed                       bool
	writeMutex                   *sync.Mutex
}

func (obcy *Obcy) Listen() {
	obcy.packetHandler = NewObcyPacketHandler()
	for {
		_, data, err := obcy.client.ReadMessage()
		if err != nil {
			if !obcy.closed {
				log.Println("Data receive failed!", err)
			}
			if obcy.strangerDisconnectedListener != nil {
				obcy.strangerDisconnectedListener()
			}
			_ = obcy.Close()
			return
		}

		//log.Println("Data received! Message:", string(data))
		packet, err := obcy.packetHandler.Parse(data)
		if err != nil {
			//log.Println("PacketHandler parse failed. Reason:", err)
			continue
		}

		err = packet.Handle(obcy)
		if err != nil {
			log.Println("PacketHandler handle failed. Reason:", err)
			continue
		}
	}
}

func (obcy *Obcy) Connect() (err error) {
	resp, err := http.Get("https://6obcy.org/rozmowa")
	if err != nil {
		return
	}
	err = resp.Body.Close()
	if err != nil {
		return
	}

	resp, err = http.Get("https://api.ipify.org/?format=raw")
	if err != nil {
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	err = resp.Body.Close()
	if err != nil {
		return
	}
	ip := string(bytes)
	ip = strings.Replace(ip, ".", "#", -1)
	fmt.Println("ip:", ip)

	port := rand.Intn(8) + 7001
	headers := http.Header{}
	http.Header.Add(headers, "Host", fmt.Sprintf("server.6obcy.pl:%d", port))
	http.Header.Add(headers, "Origin", "https://6obcy.org")
	http.Header.Add(headers, "User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/77.0.3865.90 Safari/537.36")
	websocket.DefaultDialer.EnableCompression = true
	obcy.client, _, err = websocket.DefaultDialer.Dial(fmt.Sprintf(serverAddress, port), headers)
	if err != nil {
		return
	}
	go obcy.Listen()
	go func() {
		for !obcy.closed {
			time.Sleep(30 * time.Second)
			_ = obcy.writePacket(`2`)
		}
	}()

	time.Sleep(1 * time.Second)
	err = obcy.writePacket(
		`4{"ev_name":"_cinfo","ev_data":{"cvdate":"2017-08-01","mobile":false,"cver":"v2.5","adf":"ajaxPHP","hash":"51#83#230#171","testdata":{"ckey":0,"recevsent":false}}}`)
	if err != nil {
		return
	}
	err = obcy.writePacket(`4{"ev_name":"_owack"}`)
	return
}

func (obcy *Obcy) writePacket(packet string) (err error) {
	//log.Println("Sending packet:", packet)
	obcy.writeMutex.Lock()
	err = obcy.client.WriteMessage(websocket.TextMessage, []byte(packet))
	obcy.writeMutex.Unlock()
	return
}

func (obcy *Obcy) WriteMessage(message string) (err error) {
	//4{"ev_name":"_pmsg","ev_data":{"ckey":"0:192435472_W00Wyxx2jHzGS","msg":"please respond to me","idn":4},"ceid":7}
	obcy.ceid++
	return obcy.writePacket(fmt.Sprintf(
		`4{"ev_name":"_pmsg","ev_data":{"ckey":"%s","msg":"%s","idn":%d},"ceid":%d}`,
		escapeValue(obcy.ckey), escapeValue(message), obcy.messageId, obcy.ceid))
}

func (obcy *Obcy) SearchForRetard() (err error) {
	obcy.ceid++
	return obcy.writePacket(fmt.Sprintf(
		`4{"ev_name":"_sas","ev_data":{"channel":"main","myself":{"sex":0,"loc":0},"preferences":{"sex":0,"loc":0}},"ceid":%d}`,
		obcy.ceid))
}

func (obcy *Obcy) SetTypingStatus(typing bool) (err error) {
	return obcy.writePacket(fmt.Sprintf(
		`4{"ev_name":"_mtyp","ev_data":{"ckey":"%s","val":%t}}`,
		escapeValue(obcy.ckey), typing))
}

func (obcy *Obcy) DisconnectRetard() (err error) {
	obcy.ceid++
	return obcy.writePacket(fmt.Sprintf(
		`4{"ev_name":"_distalk","ev_data":{"ckey":"%s"},"ceid":%d}`,
		escapeValue(obcy.ckey),
		obcy.ceid))
}

func (obcy *Obcy) Close() (err error) {
	obcy.closed = true
	return obcy.client.Close()
}

func (obcy *Obcy) OnMessageReceive(messageListener func(message string)) {
	obcy.messageListener = messageListener
}

func (obcy *Obcy) OnStrangerConnected(strangerConnectedListener func()) {
	obcy.strangerConnectedListener = strangerConnectedListener
}

func (obcy *Obcy) OnStrangerDisconnected(strangerDisconnectedListener func()) {
	obcy.strangerDisconnectedListener = strangerDisconnectedListener
}

func (obcy *Obcy) OnStrangerTypingStatus(strangerTypingStatus func(status bool)) {
	obcy.strangerTypingStatusListener = strangerTypingStatus
}

var globalSessionId = 0

type Obcies struct {
	sessionId          int
	disconnectListener func()
	clientOne          *Obcy
	clientTwo          *Obcy
	queuedMessages     []string
	chatHistory        []string
	chatMutex          *sync.RWMutex
	service            *ObcyService
	showMessages       bool
}

func NewObcies(service *ObcyService) *Obcies {
	globalSessionId++
	return &Obcies{
		sessionId:   globalSessionId,
		chatHistory: make([]string, 0),
		service:     service,
		chatMutex:   &sync.RWMutex{},
	}
}

func (obcies *Obcies) Connect() (err error) {
	dcChan := make(chan uint8)
	obcies.disconnectListener = func() {
		//log.Println("Obcies session closing")
		dcChan <- 1
	}

	obcies.clientOne, err = obcies.createClient()
	if err != nil {
		return fmt.Errorf("client one connect failed: %s", err.Error())
	}

	obcies.clientOne.OnStrangerConnected(func() {
		obcies.queuedMessages = make([]string, 0)
		obcies.clientOne.OnMessageReceive(func(message string) {
			obcies.queuedMessages = append(obcies.queuedMessages, message)
		})

		log.Println("Stranger connected to client one!")
		obcies.clientTwo, err = obcies.createClient()
		if err != nil {
			log.Printf("client two connect failed: %s\n", err.Error())
		}

		obcies.clientTwo.OnStrangerConnected(func() {
			log.Println("Stranger connected to client two!")
			for _, message := range obcies.queuedMessages {
				log.Println("resending message", message)
				err := obcies.clientTwo.WriteMessage(message)
				if err != nil {
					log.Printf("client two message resend failed: %s\n", err.Error())
				}
			}
			obcies.clientTwo.strangerConnectedListener = nil

			obcies.initMessageProxy()
		})
	})

	go func() {
		time.Sleep(60 * time.Second)
		obcies.chatMutex.RLock()
		if len(obcies.chatHistory) <= 2 {
			obcies.chatMutex.RUnlock()

			log.Println("Closing inactive session id:", obcies.sessionId)
			err := obcies.clientOne.DisconnectRetard()
			if err == nil {
				if obcies.clientOne.strangerDisconnectedListener != nil {
					obcies.clientOne.strangerDisconnectedListener()
				}
				obcies.service.obcyPool.Put(obcies.clientOne)
			}

			err = obcies.clientTwo.DisconnectRetard()
			if err == nil {
				if obcies.clientTwo.strangerDisconnectedListener != nil {
					obcies.clientTwo.strangerDisconnectedListener()
				}
				obcies.service.obcyPool.Put(obcies.clientTwo)
			}
		} else {
			obcies.chatMutex.RUnlock()
		}
	}()

	_ = <-dcChan
	_ = <-dcChan

	return
}

func (obcies *Obcies) initMessageProxy() {
	var clientOneName = fmt.Sprintf("[%d] jan", obcies.sessionId)
	var clientTwoName = fmt.Sprintf("[%d] karol", obcies.sessionId)

	proxies := []func(logPrefix string, clientOne, clientTwo *Obcy){
		obcies.listenConnectionStatusProxy,
		obcies.listenMessageProxy,
		obcies.listenTypeStatusProxy,
	}

	for _, proxy := range proxies {
		proxy(clientOneName, obcies.clientOne, obcies.clientTwo)
		proxy(clientTwoName, obcies.clientTwo, obcies.clientOne)
	}
}

func (obcies *Obcies) listenMessageProxy(logPrefix string, clientOne, clientTwo *Obcy) {
	clientOne.OnMessageReceive(func(message string) {
		obcies.chatMutex.Lock()
		obcies.chatHistory = append(obcies.chatHistory, fmt.Sprintf("%s: %s", logPrefix, message))
		obcies.chatMutex.Unlock()

		obcies.chatMutex.RLock()
		if !obcies.showMessages && (len(obcies.chatHistory) >= 5 || strings.Contains(message, ".")) {
			obcies.chatMutex.RUnlock()

			obcies.showMessages = true
			obcies.service.LogMessage("``WOW!!! 5 wiadomosci zostalo wyslanych!!``")
			builder := strings.Builder{}
			for _, message := range obcies.chatHistory {
				builder.WriteString(message)
				builder.WriteByte('\n')
			}
			obcies.service.LogMessage(builder.String())
		} else {
			obcies.chatMutex.RUnlock()
		}

		if obcies.showMessages {
			obcies.service.LogMessage(logPrefix + " napisał " + message)
		}

		err := clientTwo.WriteMessage(message)
		if err != nil {
			log.Println("client write message failed. Reason:", err)
			return
		}
	})
}

func (obcies *Obcies) listenTypeStatusProxy(logPrefix string, clientOne, clientTwo *Obcy) {
	clientOne.OnStrangerTypingStatus(func(status bool) {
		//if status {
		//    log.Println(logPrefix, "zaczyna pisać wiadomość")
		//} else {
		//    log.Println(logPrefix, "kończy pisać wiadomość")
		//}
		err := clientTwo.SetTypingStatus(status)
		if err != nil {
			log.Println("client typing status failed. Reason:", err)
			return
		}
	})
}

func (obcies *Obcies) listenConnectionStatusProxy(logPrefix string, clientOne, clientTwo *Obcy) {
	clientOne.OnStrangerDisconnected(func() {
		if obcies.showMessages {
			obcies.service.LogMessage(logPrefix + " rozłączył się")
		}
		defer func() {
			if obcies.disconnectListener != nil {
				obcies.disconnectListener()
			}
			obcies.service.obcyPool.Put(clientOne)
		}()

		err := clientTwo.DisconnectRetard()
		if err != nil {
			//log.Println(logPrefix, "oponent disconnect failed. Reason:", err)
			return
		}
	})
}

func (obcies *Obcies) createClient() (obcy *Obcy, err error) {
	obcy = obcies.service.obcyPool.Receive()
	if obcy == nil || obcy.closed {
		obcy = new(Obcy)
		obcy.writeMutex = &sync.Mutex{}
		err = obcy.Connect()
		if err != nil {
			err = fmt.Errorf("connect failed: %s", err.Error())
			return
		}
	} else {
		fmt.Println("got bot from pool")
	}

	err = obcy.SearchForRetard()
	if err != nil {
		err = fmt.Errorf("search for retard failed: %s", err.Error())
		return
	}

	return
}

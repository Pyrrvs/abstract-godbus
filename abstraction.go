package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nyks06/dbus"
)

//The SessionType type is a int based type used to define consts below (SESSION & SYSTEM)
type SessionType int

const (
	//SESSION Contant is used in initSession to let the user create a sessionBus
	SESSION SessionType = iota
	//SYSTEM Contant is used in initSession to let the user create a sessionBus
	SYSTEM SessionType = iota
)

//The dbusSignal type is a copy of dbus.Signal type, used to parse received signals
type dbusAbsSignal struct {
	recv    *dbus.Signal
	signame string
}

type dbusAbstraction struct {
	conn       *dbus.Conn
	recv       chan *dbus.Signal
	sigmap     map[string]chan *dbusAbsSignal
	sigsenders []string
}

//GetConn method return the current instance of *dbus.Conn
func (d *dbusAbstraction) GetConn() *dbus.Conn {
	return d.conn
}

//InitSession method is the first callable. It permits to init a session (Session or System) over the bus and request a name on it.
//Parameters :
//              s -> dbus.SessionType : equal to SESSION or SYSTEM
//              n -> string           : name you want to request over the bus (or "")
func (d *dbusAbstraction) InitSession(s SessionType, n string) error {
	var err error
	var conn *dbus.Conn

	if d.conn != nil {
		return errors.New("[DBUS ABSTRACTION ERROR - initSession - Session already initialized]")
	}

	if s == SESSION {
		conn, err = dbus.SessionBus()
	} else {
		conn, err = dbus.SystemBus()
	}
	if err != nil {
		return err
	}

	if n != "" {
		reply, err := conn.RequestName(n, dbus.NameFlagDoNotQueue)
		if err != nil {
			return err
		}
		if reply != dbus.RequestNameReplyPrimaryOwner {
			return errors.New("[DBUS ABSTRACTION ERROR - initSession - name already taken]")
		}
	}

	d.conn = conn
	d.sigmap = make(map[string]chan *dbusAbsSignal)
	d.recv = make(chan *dbus.Signal, 1024)
	go d.signalsHandler()
	return nil
}

//ExportMethods method is usable each time the user wants to export an interface over the bus
//Parameters :
//              m -> interface{}     : the interface containing the methods the user wants to export
//              p -> dbus.ObjectPath : the objectPath in which the user wants to export methods
//              i -> string          : the interface in which the user wants to export methods
func (d *dbusAbstraction) ExportMethods(m interface{}, p dbus.ObjectPath, i string) {
	d.conn.Export(m, p, i)
}

//InitSignalsListeningFor method is usable to set a new 'listener'. This listener will fill a channel each time a signal is send
//Parameters :
//              p -> string           : the ObjectPath of the sender
//              n -> string           : the name of the sender
//              i -> string           : the interface of the sender
//              s -> string           : the signal sent
func (d *dbusAbstraction) ListenSignalFromSender(p string, n string, i string, s string) {
	listened := false
	for _, elem := range d.sigsenders {
		if elem == n {
			listened = true
		}
	}
	if listened {

		//check if the entry already exist in the map [entry is sender.member]
		//yes : do nothing
		//no  : create the entry and the channel
	} else {
		_ = append(d.sigsenders, n)
		d.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, "type='signal',path='"+p+"',interface='"+i+"', sender='"+n+"'")
		//add the specific channel and entry to the map
	}
}

func (d *dbusAbstraction) getSignalName(s string) string {
	tmp := strings.Split(s, ".")
	return tmp[len(tmp)-1]
}

//signalsHandler method is called in the InitSession method. It permits to handle our signals and put them in the map
//This method run in a special goroutines. It read each signal comming from a registered sender and put it in the sigmap
func (d *dbusAbstraction) signalsHandler() {
	d.conn.Signal(d.recv)
	for v := range d.recv {

		if _, ok := d.sigmap[v.Name]; ok {
			// d.sigmap[t.signame] = make(chan *dbusSignalAbstraction, 1024)
			var t dbusAbsSignal
			t.recv = v
			t.signame = v.Name
			d.sigmap[v.Name] <- &t
			fmt.Println(v)
		}
	}
}

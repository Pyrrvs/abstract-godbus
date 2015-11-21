package main

import (
	"bytes"
	"errors"
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
	// go d.signalsHandler()
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
//Steps :
// 							We check if we already listen to this sender (if yes, the name should be in our d.sigsenders slice)
//								If we already listen to it, we check if we already listen this signal
//									If we already listen to the signal we quit, else we create the channel and the entry in the map
//								else we call the AddMatch method to listen this sender and we create the channel and the entry in the map
func (d *dbusAbstraction) ListenSignalFromSender(p string, n string, i string, s string) {
	listened := false
	for _, elem := range d.sigsenders {
		if elem == n {
			listened = true
		}
	}
	if listened {
		if _, ok := d.sigmap[d.getGeneratedName(n, s)]; !ok {
			d.sigmap[d.getGeneratedName(n, s)] = make(chan *dbusAbsSignal)
		}
	} else {
		d.sigsenders = append(d.sigsenders, n)
		d.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, "type='signal',path='"+p+"',interface='"+i+"', sender='"+n+"'")
		d.sigmap[d.getGeneratedName(n, s)] = make(chan *dbusAbsSignal, 1024)
	}
}

//CallMethod method permit to call a method over the bus. It returns nil if the method has been called and call.Err if an error occured.
//Parameters :
//              p -> dbus.ObjectPath  : the ObjectPath of the sender
//              n -> string           : the name of the sender
//              i -> string           : the interface of the sender
//              m -> string           : the method name
//							params -> string			: the method params (string for the moment)
//Response :
//The response is stored in the call struct that contains following useful fields :
// 							Args -> []interface{} : args we give in our call to the dbus method
// 							Body -> []interface{} : args we give in our call to the dbus method
// 							Err -> error          : an error variable, filled if an error occured during the call
func (d *dbusAbstraction) CallMethod(p dbus.ObjectPath, n string, i string, m string, params string) error {
	obj := d.conn.Object(n, p)
	call := obj.Call(d.getGeneratedName(i, m), 0, params)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

//Simple util method to concatenate the sender name and the method/signal name to obtain the form "sender.member"
func (d *dbusAbstraction) getGeneratedName(s string, m string) string {
	var buffer bytes.Buffer
	buffer.WriteString(s)
	buffer.WriteString(".")
	buffer.WriteString(m)
	return buffer.String()
}

//Simple util method to split the form "sender.member" and obtain the member part (split with the last dot and get the rightmost entry)
func (d *dbusAbstraction) getSignalName(s string) string {
	tmp := strings.Split(s, ".")
	return tmp[len(tmp)-1]
}

//GetSignal method return the first signal from the channel that correspond to the signal given as parameter
//Parameters :
//              s -> string  : signal you want to get
func (d *dbusAbstraction) GetSignal(s string) ([]interface{}, error) {
	if _, ok := d.sigmap[s]; ok {
		t := <-d.sigmap[s]
		return t.recv.Body, nil
	}
	return nil, errors.New("[DBUS ABSTRACTION] - error - not listened signal")
}

//signalsHandler method is called in the InitSession method. It permits to handle our signals and put them in the map
//This method run in a special goroutines. It read each signal comming from a registered sender and put it in the sigmap
func (d *dbusAbstraction) signalsHandler() {
	d.conn.Signal(d.recv)
	for v := range d.recv {
		if _, ok := d.sigmap[v.Name]; ok {
			var t dbusAbsSignal
			t.recv = v
			t.signame = v.Name
			d.sigmap[v.Name] <- &t
		}
	}
}

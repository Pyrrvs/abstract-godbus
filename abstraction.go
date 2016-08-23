package abstractdbus

import (
	"bytes"
	"errors"
	"strings"

	"github.com/Nyks06/dbus"
)

//##################
//## TYPES AND VARS
//##################

//The SessionType type is a int based type used to define consts below (SESSION & SYSTEM)
type SessionType int

const (
	//SESSION Contant is used in initSession to let the user create a sessionBus
	SESSION SessionType = iota
	//SYSTEM Contant is used in initSession to let the user create a sessionBus
	SYSTEM SessionType = iota
)

//AbsSignal type is a copy of dbus.Signal type, used to parse received signals
type AbsSignal struct {
	Recv    *dbus.Signal
	Signame string
}

//Abstraction type contains the necessary vars and is used as receiver of our methods
type Abstraction struct {
	Conn       *dbus.Conn
	Recv       chan *dbus.Signal
	Sigmap     map[string]chan *AbsSignal
	Sigsenders []string
}

//GetConn method return the current instance of *dbus.Conn
func (d *Abstraction) GetConn() *dbus.Conn {
	return d.Conn
}

//##################
//## INIT
//##################

//New function permit to initialize a new pointer to Abstraction used after ...
func New() *Abstraction {
	return &Abstraction{}
}

//InitSession method is the first callable. It permits to init a session (Session or System) over the bus and request a name on it.
//Parameters :
//              s -> dbus.SessionType : equal to SESSION or SYSTEM
//              n -> string           : name you want to request over the bus (or "")
func (d *Abstraction) InitSession(s SessionType, n string) error {
	var err error
	var conn *dbus.Conn

	if d.Conn != nil {
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

	d.Conn = conn
	d.Sigmap = make(map[string]chan *AbsSignal)
	d.Recv = make(chan *dbus.Signal, 1024)
	go d.signalsHandler()
	return nil
}

//##################
//## UTILS
//##################

//Simple util method to concatenate the sender name and the method/signal name to obtain the form "sender.member"
func (d *Abstraction) getGeneratedName(s string, m string) string {
	var buffer bytes.Buffer
	buffer.WriteString(s)
	buffer.WriteString(".")
	buffer.WriteString(m)
	return buffer.String()
}

//Simple util method to split the form "sender.member" and obtain the member part (split with the last dot and get the rightmost entry)
func (d *Abstraction) getSignalName(s string) string {
	tmp := strings.Split(s, ".")
	return tmp[len(tmp)-1]
}

//##################
//## GETTERS
//##################

//GetSignal method return the first signal from the channel that correspond to the signal given as parameter
//Parameters :
//              s -> string  : signal you want to get
func (d *Abstraction) GetSignal(s string) ([]interface{}, error) {
	if _, ok := d.Sigmap[s]; ok {
		t := <-d.Sigmap[s]
		return t.Recv.Body, nil
	}
	return nil, errors.New("[DBUS ABSTRACTION] - error - not listened signal")
}

//GetChannel method return the channel associated to the signal the user give as parameter
//Parameters :
//              s -> string  : signal corresponding to the channel you want to listen
func (d *Abstraction) GetChannel(s string) chan *AbsSignal {
	if _, ok := d.Sigmap[s]; ok {
		return d.Sigmap[s]
	}
	return nil
}

//##################
//## METHODS MANAGEMENT
//##################

//ExportMethods method is usable each time the user wants to export an interface over the bus
//Parameters :
//              m -> interface{}     : the interface containing the methods the user wants to export
//              p -> dbus.ObjectPath : the objectPath in which the user wants to export methods
//              i -> string          : the interface in which the user wants to export methods
func (d *Abstraction) ExportMethods(m interface{}, p dbus.ObjectPath, i string) {
	d.Conn.Export(m, p, i)
}

//CallMethod method permit to call a method over the bus. It returns nil if the method has been called and call.Err if an error occured.
//Parameters :
//              p -> dbus.ObjectPath  : the ObjectPath of the sender
//              n -> string           : the name of the sender
//              i -> string           : the interface of the sender
//              m -> string           : the method name
//		params -> string      : the method params (string for the moment)
//Response :
//The response is stored in the call struct that contains following useful fields :
// 		Args -> []interface{} : args we give in our call to the dbus method
// 		Body -> []interface{} : args we give in our call to the dbus method
// 		Err -> error          : an error variable, filled if an error occured during the call
func (d *Abstraction) CallMethod(p dbus.ObjectPath, n string, i string, m string, params string) error {
	obj := d.Conn.Object(n, p)
	call := obj.Call(d.getGeneratedName(i, m), 0, params)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

//##################
//## SIGNALS MANAGEMENT
//##################

//ListenSignalFromSender method is usable to set a new 'listener'. This listener will fill a channel each time a signal is send
//Parameters :
//              p -> string           : the ObjectPath of the sender
//              n -> string           : the name of the sender
//              i -> string           : the interface of the sender
//              s -> string           : the signal sent
//Steps :
// 		we check if we already listen to this sender (if yes, the name should be in our d.sigsenders slice)
//		If we already listen to it, we check if we already listen this signal
//		Else if we already listen to the signal we quit, else we create the channel and the entry in the map
//		else we call the AddMatch method to listen this sender and we create the channel and the entry in the map
func (d *Abstraction) ListenSignalFromSender(p string, n string, i string, s string) {
	listened := false
	for _, elem := range d.Sigsenders {
		if elem == n {
			listened = true
		}
	}
	if listened {
		if _, ok := d.Sigmap[d.getGeneratedName(n, s)]; !ok {
			d.Sigmap[d.getGeneratedName(n, s)] = make(chan *AbsSignal)
		}
	} else {
		d.Sigsenders = append(d.Sigsenders, n)
		d.Conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, "type='signal',path='"+p+"',interface='"+i+"', sender='"+n+"'")
		d.Sigmap[d.getGeneratedName(n, s)] = make(chan *AbsSignal, 1024)
	}
}

//signalsHandler method is called in the InitSession method. It permits to handle our signals and put them in the map
//This method run in a special goroutines. It read each signal comming from a registered sender and put it in the sigmap
func (d *Abstraction) signalsHandler() {
	d.Conn.Signal(d.Recv)
	for v := range d.Recv {
		if _, ok := d.Sigmap[v.Name]; ok {
			var t AbsSignal
			t.Recv = v
			t.Signame = v.Name
			d.Sigmap[v.Name] <- &t
		}
	}
}

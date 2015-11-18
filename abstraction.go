package main

import (
	"errors"

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

type dbusAbstraction struct {
	conn *dbus.Conn
	sign map[string]chan *dbus.Signal
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
	d.sign = make(map[string]chan *dbus.Signal)

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

//ListenSignals method is usable to set a new 'listener'. This listener will fill a channel each time a signal is send
//Parameters :
//              p -> string           : the ObjectPath of the sender
//              n -> string           : the name of the sender
//              i -> string           : the interface of the sender

func (d *dbusAbstraction) InitSignalsListeningFor(p string, n string, i string) {
	d.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, "type='signal',path='"+p+"',interface='"+i+"', sender='"+n+"'")
}

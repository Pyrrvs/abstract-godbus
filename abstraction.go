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
}

//InitSession method is the first callable. It permits to init a session (Session or System) over the bus and request a name on it.
//Parameters :
//              s -> dbus.SessionType : equal to SESSION or SYSTEM
//              n -> string           : name you want to request over the bus (or "")
func (d *dbusAbstraction) InitSession(s SessionType, n string) error {
	var err error
	var conn *dbus.Conn

	if s == SESSION {
		conn, err = dbus.SessionBus()
	} else {
		conn, err = dbus.SystemBus()
	}
	if err != nil {
		return err
	}
	d.conn = conn

	if n != "" {
		reply, err := d.conn.RequestName(n, dbus.NameFlagDoNotQueue)
		if err != nil {
			return err
		}
		if reply != dbus.RequestNameReplyPrimaryOwner {
			return errors.New("[DBUS ABSTRACTION ERROR - initSession - name already taken]")
		}
	}
	return nil
}

//ExportMethods method is usable each time the user want to export an interface over the bus
//Parameters :
//              m -> interface{}     : the interface containing the methods the user wants to export
//              p -> dbus.ObjectPath : the objectPath in which the user wants to export methods
//              i -> string          : the interface in which the user wants to export methods
func (d *dbusAbstraction) ExportMethods(m interface{}, p dbus.ObjectPath, i string) {
	d.conn.Export(m, p, i)
}

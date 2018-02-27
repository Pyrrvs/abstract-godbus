package AbstractDBus

import "github.com/Pyrrvs/dbus"

func GetDbus() (*dbus.Conn, error) {
  return dbus.SystemBus()
}
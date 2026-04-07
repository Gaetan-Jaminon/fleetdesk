package main

// Type aliases bridging root package to internal/config.
// These will be removed when all code moves to internal/.

import "github.com/Gaetan-Jaminon/fleetdesk/internal/config"

type fleet = config.Fleet
type hostDefaults = config.HostDefaults
type hostGroup = config.HostGroup
type hostEntry = config.HostEntry
type host = config.Host
type hostStatus = config.HostStatus

const (
	hostConnecting  = config.HostConnecting
	hostOnline      = config.HostOnline
	hostUnreachable = config.HostUnreachable
)

type service = config.Service
type container = config.Container
type cronJob = config.CronJob
type logLevelEntry = config.LogLevelEntry
type errorLog = config.ErrorLog
type update = config.Update
type subscription = config.Subscription
type disk = config.Disk

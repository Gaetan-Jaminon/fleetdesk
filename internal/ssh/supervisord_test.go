package ssh

import (
	"testing"
)

func TestParseSupervisordStatus_Running(t *testing.T) {
	output := `tower-processes:awx-uwsgi                RUNNING   pid 1007, uptime 0:05:45`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 1 {
		t.Fatalf("got %d procs, want 1", len(procs))
	}
	p := procs[0]
	if p.Name != "tower-processes:awx-uwsgi" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.State != "RUNNING" {
		t.Errorf("State = %q", p.State)
	}
	if p.PID != "1007" {
		t.Errorf("PID = %q, want 1007", p.PID)
	}
	if p.Uptime != "0:05:45" {
		t.Errorf("Uptime = %q, want 0:05:45", p.Uptime)
	}
}

func TestParseSupervisordStatus_Fatal(t *testing.T) {
	output := `tower-processes:awx-daphne               FATAL     Exited too quickly (process log may have details)`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 1 {
		t.Fatalf("got %d procs, want 1", len(procs))
	}
	p := procs[0]
	if p.State != "FATAL" {
		t.Errorf("State = %q, want FATAL", p.State)
	}
	if p.PID != "-" {
		t.Errorf("PID = %q, want -", p.PID)
	}
}

func TestParseSupervisordStatus_Stopped(t *testing.T) {
	output := `tower-processes:awx-dispatcher           STOPPED   Not started`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 1 {
		t.Fatalf("got %d procs, want 1", len(procs))
	}
	if procs[0].State != "STOPPED" {
		t.Errorf("State = %q, want STOPPED", procs[0].State)
	}
}

func TestParseSupervisordStatus_Backoff(t *testing.T) {
	output := `tower-processes:awx-rsyslogd             BACKOFF   Exited too quickly (process log may have details)`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 1 {
		t.Fatalf("got %d procs, want 1", len(procs))
	}
	if procs[0].State != "BACKOFF" {
		t.Errorf("State = %q, want BACKOFF", procs[0].State)
	}
}

func TestParseSupervisordStatus_MultipleProcesses(t *testing.T) {
	output := `master-event-listener                    FATAL     can't find command '/usr/bin/failure-event-handler'
tower-processes:awx-callback-receiver    RUNNING   pid 10355, uptime 0:02:14
tower-processes:awx-daphne               RUNNING   pid 11090, uptime 0:00:05
tower-processes:awx-dispatcher           RUNNING   pid 10354, uptime 0:02:14
tower-processes:awx-rsyslogd             BACKOFF   Exited too quickly (process log may have details)
tower-processes:awx-uwsgi                RUNNING   pid 10356, uptime 0:02:14`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 6 {
		t.Fatalf("got %d procs, want 6", len(procs))
	}
	// Check first is FATAL
	if procs[0].State != "FATAL" {
		t.Errorf("first proc State = %q, want FATAL", procs[0].State)
	}
	// Check a running proc has PID
	if procs[1].PID != "10355" {
		t.Errorf("callback-receiver PID = %q, want 10355", procs[1].PID)
	}
}

func TestParseSupervisordStatus_Empty(t *testing.T) {
	procs := ParseSupervisordStatus("")
	if len(procs) != 0 {
		t.Errorf("got %d procs, want 0", len(procs))
	}
}

func TestParseSupervisordStatus_UngroupedName(t *testing.T) {
	output := `myprocess                                RUNNING   pid 42, uptime 1:00:00`
	procs := ParseSupervisordStatus(output)
	if len(procs) != 1 {
		t.Fatalf("got %d procs, want 1", len(procs))
	}
	if procs[0].Name != "myprocess" {
		t.Errorf("Name = %q, want myprocess", procs[0].Name)
	}
}

func TestProcessStateOrder(t *testing.T) {
	if ProcessStateOrder("FATAL") >= ProcessStateOrder("BACKOFF") {
		t.Error("FATAL should sort before BACKOFF")
	}
	if ProcessStateOrder("BACKOFF") >= ProcessStateOrder("STOPPED") {
		t.Error("BACKOFF should sort before STOPPED")
	}
	if ProcessStateOrder("STOPPED") >= ProcessStateOrder("RUNNING") {
		t.Error("STOPPED should sort before RUNNING")
	}
	if ProcessStateOrder("RUNNING") >= ProcessStateOrder("UNKNOWN_STATE") {
		t.Error("RUNNING should sort before unknown")
	}
}

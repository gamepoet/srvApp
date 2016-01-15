//  ---------------------------------------------------------------------------
//
//  run_win.go
//
//  Copyright (c) 2015, Jared Chavez. 
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

// +build windows

package srvApp

import (
    "golang.org/x/sys/windows/svc"
    "golang.org/x/sys/windows/svc/mgr"
    "github.com/xaevman/app"
)

const accept_cmds = svc.AcceptStop | svc.AcceptShutdown

// appSvc implements the required interfaces to enable the application
// to run as a windows service and be controlled by the windows service
// control manager.
type appSvc struct {}

// Execute is the entry point of execution for a windows service.
func (this *appSvc) Execute(
    args    []string, 
    r       <-chan svc.ChangeRequest, 
    changes chan<- svc.Status,
) (
    ssec  bool, 
    errno uint32,
) {
    Log().Info("Initializing service")

    changes<- svc.Status {
        State : svc.StartPending,
    }

    if !startSingleton() {
        Log().Error("Failed to start Singleton")
        changes<- svc.Status {
            State : svc.StopPending,
        }

        signalShutdown()
    }

    Log().Info("Service initialized")
    changes<- svc.Status {
        Accepts : accept_cmds,
        State   : svc.Running,
    }

    for !shuttingDown {
        select {
        case <-shutdownChan:
            Log().Debug("Service shutdown signal received")
            shutdown()
        case c := <-r:
            switch c.Cmd {
            case svc.Interrogate:
                Log().Debug("Service interrogate: %v", c)
                changes<- c.CurrentStatus
            case svc.Stop, svc.Shutdown:
                Log().Info("Service stop requested")
                changes<- svc.Status {
                    State : svc.StopPending,
                }
                signalShutdown()
            default:
                Log().Error(
                    "Unhandled signal received from SCM: %v", 
                    c,
                )
            }
        }
    }

    changes<- svc.Status {
        State : svc.Stopped,
    }

    return
}

// afterFlags captures the service install and uninstall run modes
// and executes them, if needed, after command-line flags are parsed.
func afterFlags() {
    switch runMode {
    case INST_SVC:
        installSvc()
    case UNINST_SVC:
        uninstallSvc()
    }
}

// installSvc attempts to install the running binary as a windows service.
func installSvc() {
    defer signalShutdown()
    
    Log().Info("Installing service %s", app.GetName())
    scm, err := mgr.Connect()
    if err != nil {
        Log().Error("Error connecting to SCM: %v", err)
        return
    }

    defer scm.Disconnect()

    svc, err := scm.OpenService(app.GetName())
    if err == nil {
        Log().Error("Service already exists")
        return
    }

    svc, err = scm.CreateService(
        app.GetName(),
        app.GetExePath(),
        mgr.Config{
            StartType : mgr.StartAutomatic,
        },
        "-runSvc",
    )
    defer svc.Close()

    if err != nil {
        Log().Error("Error creating service: %v", err)
        return
    }

    Log().Info("Service %s installed", app.GetName())
}

// run executes the application in either console or service run mode,
// depending on the arguments supplied on the command line.
func run() {
    switch runMode {
    case CMDLINE:
        runCmdline()
    case RUN_SVC:
        runSvc()
    }
}

// runCmdLine runs the application in console mode.
func runCmdline() {
    if !startSingleton() {
        signalShutdown()
    }

    blockUntilShutdown()
}

// runSvc starts the application in service mode.
func runSvc() {
    err := svc.Run(app.GetName(), &appSvc{})
    if err != nil {
        Log().Error("Service execution error: %v", err)
        signalShutdown()
    }
}

// uninstallSvc attempts to uninstall the running binary from the service
// control manager.
func uninstallSvc() {
    defer signalShutdown()

    Log().Info("Removing service %s", app.GetName())
    scm, err := mgr.Connect()
    if err != nil {
        Log().Error("Error connecting to SCM: %v", err)
        return
    }

    defer scm.Disconnect()

    svc, err := scm.OpenService(app.GetName())
    if err != nil {
        Log().Error("Service %s doesn't exist", app.GetName())
        return
    }

    defer svc.Close()
    err = svc.Delete()
    if err != nil {
        Log().Error("Error deleting service: %v", err)
        return
    }

    Log().Info("Service %s deleted", app.GetName())
}

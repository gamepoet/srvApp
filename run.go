//  ---------------------------------------------------------------------------
//
//  run.go
//
//  Copyright (c) 2015, Jared Chavez. 
//  All rights reserved.
//
//  Use of this source code is governed by a BSD-style
//  license that can be found in the LICENSE file.
//
//  -----------

// +build !windows

package srvApp

func afterFlags() {}

func run() {
    if initComplete {
        if !startSingleton() {
            shutdownChan<- true
        }

        blockUntilShutdown()
    }
}

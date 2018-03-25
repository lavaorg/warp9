// Copyright 2009 The Go9p Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package warp9

import (
	"errors"
	"strings"
)

// Opens the file associated with the fid. Returns nil if
// the operation is successful.
func (clnt *Clnt) Open(fid *Fid, mode uint8) error {
	tc := clnt.NewFcall()
	err := tc.packTopen(fid.Fid, mode)
	if err != nil {
		return err
	}

	rc, err := clnt.Rpc(tc)
	if err != nil {
		return err
	}

	fid.Qid = rc.Qid
	fid.Iounit = rc.Iounit
	if fid.Iounit == 0 || fid.Iounit > clnt.Msize-IOHDRSZ {
		fid.Iounit = clnt.Msize - IOHDRSZ
	}
	fid.Mode = mode
	return nil
}

// Creates a file in the directory associated with the fid. Returns nil
// if the operation is successful.
func (clnt *Clnt) Create(fid *Fid, name string, perm uint32, mode uint8, extattr string) error {
	tc := clnt.NewFcall()
	err := tc.packTcreate(fid.Fid, name, perm, mode, extattr)
	if err != nil {
		return err
	}

	rc, err := clnt.Rpc(tc)
	if err != nil {
		return err
	}

	fid.Qid = rc.Qid
	fid.Iounit = rc.Iounit
	if fid.Iounit == 0 || fid.Iounit > clnt.Msize-IOHDRSZ {
		fid.Iounit = clnt.Msize - IOHDRSZ
	}
	fid.Mode = mode
	return nil
}

// Creates and opens a named file.
// Returns the file if the operation is successful, or an Error.
func (clnt *Clnt) FCreate(path string, perm uint32, mode uint8) (*File, error) {
	n := strings.LastIndex(path, "/")
	if n < 0 {
		n = 0
	}

	fid, err := clnt.FWalk(path[0:n])
	if err != nil {
		return nil, err
	}

	if path[n] == '/' {
		n++
	}

	err = clnt.Create(fid, path[n:], perm, mode, "")
	if err != nil {
		clnt.Clunk(fid)
		return nil, err
	}

	return &File{fid, 0}, nil
}

// Opens a named file. Returns the opened file, or an Error.
func (clnt *Clnt) FOpen(path string, mode uint8) (*File, error) {
	fid, err := clnt.FWalk(path)
	if err != nil {
		return nil, err
	}

	err = clnt.Open(fid, mode)
	if err != nil {
		clnt.Clunk(fid)
		return nil, err
	}

	return &File{fid, 0}, nil
}

// Opens an existing fid, returns a valid File. This behaves as does
// FOpen, however, using a previously aquired fid, usually from FWalk.
func (clnt *Clnt) FFidOpen(fid *Fid, mode uint8) (*File, error) {
	if fid == nil {
		return nil, errors.New("nil fid")
	}
	if !fid.walked {
		return nil, errors.New("fid not from valid walk")
	}

	err := clnt.Open(fid, mode)
	if err != nil {
		clnt.Clunk(fid)
		return nil, err
	}

	return &File{fid, 0}, nil

}

// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package warpsrv

import (
	"sync"
	"time"

	"github.com/lavaorg/warp/warp9"
)

type FFlags int

const (
	Fremoved FFlags = 1 << iota
)

// The srvFile type represents a file (or directory) served by the file server.
type W9File struct {
	sync.Mutex
	warp9.Dir
	flags FFlags

	Parent *W9File // parent

	next, prev    *W9File // siblings, guarded by parent.Lock
	cfirst, clast *W9File // children (if directory)
	ops           interface{}
}

// A server representation of a client
type W9Fid struct {
	F       *W9File
	Fid     *warp9.SrvFid
	dirs    []*W9File // used for readdir
	dirents []byte    // serialized version of dirs
}

// W9Srv provides a framework for creating synthetic file systems. The file system
// is in memory, active, and can be multi-level.
type W9Srv struct {
	warp9.Srv
	Root *W9File
}

var lock sync.Mutex
var qnext uint64

// Creates a file server with root as root directory
func NewW9Srv(root *W9File) *W9Srv {
	srv := new(W9Srv)
	srv.Root = root
	root.Parent = root // make sure we can .. in root

	return srv
}

// Initializes the fields of a file and add it to a directory.
// Returns nil if successful, or an error.
func (f *W9File) Add(dir *W9File, name string, uid warp9.User, gid warp9.Group, mode uint32, ops interface{}) error {

	lock.Lock()
	qpath := qnext
	qnext++
	lock.Unlock()

	f.Qid.Type = uint8(mode >> 24)
	f.Qid.Version = 0
	f.Qid.Path = qpath
	f.Mode = mode
	f.Atime = uint32(time.Now().Unix())
	f.Mtime = f.Atime
	f.Length = 0
	f.Name = name
	if uid != nil {
		f.Uid = uid.Name()
	} else {
		f.Uid = "none"
	}

	if gid != nil {
		f.Gid = gid.Name()
	} else {
		f.Gid = "none"
	}

	f.Muid = ""
	f.ExtAttr = ""

	// add f as entry in dir
	if dir != nil {
		f.Parent = dir
		dir.Lock()
		for p := dir.cfirst; p != nil; p = p.next {
			if name == p.Name {
				dir.Unlock()
				return warp9.Err(warp9.Eexist)
			}
		}

		if dir.clast != nil {
			dir.clast.next = f
		} else {
			dir.cfirst = f
		}

		f.prev = dir.clast
		f.next = nil
		dir.clast = f
		dir.Unlock()
	} else {
		f.Parent = f
	}

	f.ops = ops
	return nil
}

// Removes a file from its parent directory.
func (f *W9File) Remove() {
	f.Lock()
	if (f.flags & Fremoved) != 0 {
		f.Unlock()
		return
	}

	f.flags |= Fremoved
	f.Unlock()

	p := f.Parent
	p.Lock()
	if f.next != nil {
		f.next.prev = f.prev
	} else {
		p.clast = f.prev
	}

	if f.prev != nil {
		f.prev.next = f.next
	} else {
		p.cfirst = f.next
	}

	f.next = nil
	f.prev = nil
	p.Unlock()
}

func (f *W9File) Rename(name string) error {
	p := f.Parent
	p.Lock()
	defer p.Unlock()
	for c := p.cfirst; c != nil; c = c.next {
		if name == c.Name {
			return warp9.Err(warp9.Eexist)
		}
	}

	f.Name = name
	return nil
}

// Looks for a file in a directory. Returns nil if the file is not found.
func (p *W9File) Find(name string) *W9File {
	var f *W9File

	p.Lock()
	for f = p.cfirst; f != nil; f = f.next {
		if name == f.Name {
			break
		}
	}
	p.Unlock()
	return f
}

// Checks if the specified user has permission to perform
// certain operation on a file. Perm contains one or more
// of DMREAD, DMWRITE, and DMEXEC.
func (f *W9File) CheckPerm(user warp9.User, perm uint32) bool {
	if user == nil {
		return false
	}

	perm &= 7 //ignore non perm-bits

	/* other permissions */
	fperm := f.Mode & 7
	if (fperm & perm) == perm {
		return true
	}

	/* user permissions */
	if f.Uid == user.Name() {
		fperm |= (f.Mode >> 6) & 7
	}

	if (fperm & perm) == perm {
		return true
	}

	/* group permissions */
	groups := user.Groups()
	if groups != nil && len(groups) > 0 {
		for i := 0; i < len(groups); i++ {
			if f.Gid == groups[i].Name() {
				fperm |= (f.Mode >> 3) & 7
				break
			}
		}
	}

	if (fperm & perm) == perm {
		return true
	}

	return false
}

func mode2Perm(mode uint8) uint32 {
	var perm uint32 = 0

	switch mode & 3 {
	case warp9.OREAD:
		perm = warp9.DMREAD
	case warp9.OWRITE:
		perm = warp9.DMWRITE
	case warp9.ORDWR:
		perm = warp9.DMREAD | warp9.DMWRITE
	}

	if (mode & warp9.OTRUNC) != 0 {
		perm |= warp9.DMWRITE
	}

	return perm
}

func (*W9Srv) FidDestroy(ffid *warp9.SrvFid) {
	if ffid.Aux == nil {
		return
	}
	fid := ffid.Aux.(*W9Fid)
	f := fid.F

	if f == nil {
		return // otherwise errs in bad walks
	}

	if op, ok := (f.ops).(FDestroyOp); ok {
		op.FidDestroy(fid)
	}
}

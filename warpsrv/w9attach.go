// Copyright 2018 Larry Rau. All rights reserved
// See Apache2 LICENSE

package warpsrv

import "github.com/lavaorg/warp9/warp9"

func (s *W9Srv) Attach(req *warp9.SrvReq) {
	fid := new(W9Fid)
	fid.F = s.Root
	fid.Fid = req.Fid
	req.Fid.Aux = fid
	req.RespondRattach(&s.Root.Qid)
}

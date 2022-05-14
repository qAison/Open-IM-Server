package cache

import (
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/constant"
	"Open_IM/pkg/common/db"
	imdb "Open_IM/pkg/common/db/mysql_model/im_mysql_model"
	"Open_IM/pkg/common/log"
	"Open_IM/pkg/grpc-etcdv3/getcdv3"
	pbCache "Open_IM/pkg/proto/cache"
	commonPb "Open_IM/pkg/proto/sdk_ws"
	"Open_IM/pkg/utils"
	"context"
	"net"
	"strconv"
	"strings"

	"google.golang.org/grpc"
)

type cacheServer struct {
	rpcPort         int
	rpcRegisterName string
	etcdSchema      string
	etcdAddr        []string
}

func NewCacheServer(port int) *cacheServer {
	log.NewPrivateLog(constant.LogFileName)
	return &cacheServer{
		rpcPort:         port,
		rpcRegisterName: config.Config.RpcRegisterName.OpenImCacheName,
		etcdSchema:      config.Config.Etcd.EtcdSchema,
		etcdAddr:        config.Config.Etcd.EtcdAddr,
	}
}

func (s *cacheServer) Run() {
	log.NewInfo("0", "cacheServer rpc start ")
	listenIP := ""
	if config.Config.ListenIP == "" {
		listenIP = "0.0.0.0"
	} else {
		listenIP = config.Config.ListenIP
	}
	address := listenIP + ":" + strconv.Itoa(s.rpcPort)
	//listener network
	listener, err := net.Listen("tcp", address)
	if err != nil {
		panic("listening err:" + err.Error() + s.rpcRegisterName)
	}
	log.NewInfo("0", "listen network success, ", address, listener)
	defer listener.Close()
	//grpc server

	//to cache
	err = SyncDB2Cache()
	if err != nil {
		log.NewError("", err.Error(), "db to cache failed")
		panic(err.Error())
	}

	srv := grpc.NewServer()
	defer srv.GracefulStop()
	pbCache.RegisterCacheServer(srv, s)

	rpcRegisterIP := ""
	if config.Config.RpcRegisterIP == "" {
		rpcRegisterIP, err = utils.GetLocalIP()
		if err != nil {
			log.Error("", "GetLocalIP failed ", err.Error())
		}
	}

	err = getcdv3.RegisterEtcd(s.etcdSchema, strings.Join(s.etcdAddr, ","), rpcRegisterIP, s.rpcPort, s.rpcRegisterName, 10)
	if err != nil {
		log.NewError("0", "RegisterEtcd failed ", err.Error())
		return
	}
	err = srv.Serve(listener)
	if err != nil {
		log.NewError("0", "Serve failed ", err.Error())
		return
	}
	log.NewInfo("0", "message cms rpc success")
}

func SyncDB2Cache() error {
	var err error
	log.NewInfo("0", utils.GetSelfFuncName())
	userList, err := imdb.GetAllUser()
	log.NewDebug("", utils.GetSelfFuncName(), "userList", userList)
	if err != nil {
		return utils.Wrap(err, "")
	}
	//err = updateAllUserToCache(userList)
	err = updateAllFriendToCache(userList)
	err = updateAllBlackListToCache(userList)
	err = updateAllGroupMemberListToCache()
	return utils.Wrap(err, "")
}

func DelRelationCache() {}

func updateAllUserToCache(userList []db.User) error {
	for _, userInfo := range userList {
		userInfoPb := &commonPb.UserInfo{
			UserID:         userInfo.UserID,
			Nickname:       userInfo.Nickname,
			FaceURL:        userInfo.FaceURL,
			Gender:         userInfo.Gender,
			PhoneNumber:    userInfo.PhoneNumber,
			Birth:          uint32(userInfo.Birth.Unix()),
			Email:          userInfo.Email,
			Ex:             userInfo.Ex,
			CreateTime:     uint32(userInfo.CreateTime.Unix()),
			AppMangerLevel: userInfo.AppMangerLevel,
		}
		m, err := utils.Pb2Map(userInfoPb)
		if err != nil {
			log.NewWarn("", utils.GetSelfFuncName(), err.Error())
		}
		if err := db.DB.SetUserInfoToCache(userInfo.UserID, m); err != nil {
			log.NewWarn("0", utils.GetSelfFuncName(), "set userInfo to cache failed", err.Error())
		}
	}
	log.NewInfo("0", utils.GetSelfFuncName(), "ok")
	return nil
}

func updateAllGroupMemberListToCache() error {
	log.NewInfo("0", utils.GetSelfFuncName())
	groupIDList, err := imdb.GetAllGroupIDList()
	if err != nil {
		log.NewWarn("0", utils.GetSelfFuncName(), "getAllGroupIDList failed", err.Error())
		panic(err.Error())
	}
	for _, groupID := range groupIDList {
		groupMemberIDList, err := imdb.GetGroupMemberIDListByGroupID(groupID)
		if err != nil {
			log.NewWarn("", utils.GetSelfFuncName(), "GetGroupMemberIDListByGroupID", err.Error())
			continue
		}
		//log.NewDebug("", utils.GetSelfFuncName(), "groupMemberIDList", groupMemberIDList)
		if len(groupMemberIDList) > 0 {
			if err := db.DB.AddGroupMemberToCache(groupID, groupMemberIDList...); err != nil {
				log.NewWarn("", utils.GetSelfFuncName(), "AddGroupMemberToCache", err.Error())
			}
		}
	}
	log.NewInfo("0", utils.GetSelfFuncName(), "ok")
	return nil
}

func updateAllFriendToCache(userList []db.User) error {
	log.NewInfo("0", utils.GetSelfFuncName())
	for _, user := range userList {
		friendIDList, err := imdb.GetFriendIDListByUserID(user.UserID)
		if err != nil {
			log.NewWarn("0", utils.GetSelfFuncName(), err.Error())
			continue
		}
		if len(friendIDList) > 0 {
			if err := db.DB.AddFriendToCache(user.UserID, friendIDList...); err != nil {
				log.NewWarn("0", utils.GetSelfFuncName(), err.Error(), friendIDList, user.UserID)
			}
		}
	}
	log.NewInfo("0", utils.GetSelfFuncName(), "ok")
	return nil
}

func updateAllBlackListToCache(userList []db.User) error {
	log.NewInfo("0", utils.GetSelfFuncName())
	for _, user := range userList {
		blackIDList, err := imdb.GetBlackIDListByUserID(user.UserID)
		if err != nil {
			log.NewWarn("", utils.GetSelfFuncName(), err.Error())
			continue
		}
		if len(blackIDList) > 0 {
			if err := db.DB.AddBlackUserToCache(user.UserID, blackIDList...); err != nil {
				log.NewWarn("0", utils.GetSelfFuncName(), err.Error())
			}
		}
	}
	log.NewInfo("0", utils.GetSelfFuncName(), "ok")
	return nil
}

func (s *cacheServer) GetUserInfoFromCache(_ context.Context, req *pbCache.GetUserInfoFromCacheReq) (resp *pbCache.GetUserInfoFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.GetUserInfoFromCacheResp{
		CommonResp: &pbCache.CommonResp{},
	}
	for _, userID := range req.UserIDList {
		userInfo, err := db.DB.GetUserInfoFromCache(userID)
		if err != nil {
			log.NewError(req.OperationID, utils.GetSelfFuncName(), "get userInfo from cache failed", err.Error())
			continue
		}
		resp.UserInfoList = append(resp.UserInfoList, userInfo)
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) UpdateUserInfoToCache(_ context.Context, req *pbCache.UpdateUserInfoToCacheReq) (resp *pbCache.UpdateUserInfoToCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.UpdateUserInfoToCacheResp{
		CommonResp: &pbCache.CommonResp{},
	}
	for _, userInfo := range req.UserInfoList {
		m, err := utils.Pb2Map(userInfo)
		if err != nil {
			log.NewError(req.OperationID, utils.GetSelfFuncName(), err.Error(), *userInfo)
		}
		if err := db.DB.SetUserInfoToCache(userInfo.UserID, m); err != nil {
			log.NewError(req.OperationID, utils.GetSelfFuncName(), "set userInfo to cache failed", err.Error())
		}
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) GetFriendIDListFromCache(_ context.Context, req *pbCache.GetFriendIDListFromCacheReq) (resp *pbCache.GetFriendIDListFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.GetFriendIDListFromCacheResp{CommonResp: &pbCache.CommonResp{}}
	friendIDList, err := db.DB.GetFriendIDListFromCache(req.UserID)
	if err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "GetFriendIDListFromCache", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewDebug(req.OperationID, utils.GetSelfFuncName(), friendIDList)
	resp.UserIDList = friendIDList
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) AddFriendToCache(_ context.Context, req *pbCache.AddFriendToCacheReq) (resp *pbCache.AddFriendToCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.AddFriendToCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.AddFriendToCache(req.UserID, req.FriendID); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "AddFriendToCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) ReduceFriendFromCache(_ context.Context, req *pbCache.ReduceFriendFromCacheReq) (resp *pbCache.ReduceFriendFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.ReduceFriendFromCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.ReduceFriendToCache(req.UserID, req.FriendID); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "AddFriendToCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) GetBlackIDListFromCache(_ context.Context, req *pbCache.GetBlackIDListFromCacheReq) (resp *pbCache.GetBlackIDListFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.GetBlackIDListFromCacheResp{CommonResp: &pbCache.CommonResp{}}
	blackUserIDList, err := db.DB.GetBlackListFromCache(req.UserID)
	if err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "AddFriendToCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewDebug(req.OperationID, utils.GetSelfFuncName(), blackUserIDList)
	resp.UserIDList = blackUserIDList
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) AddBlackUserToCache(_ context.Context, req *pbCache.AddBlackUserToCacheReq) (resp *pbCache.AddBlackUserToCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.AddBlackUserToCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.AddBlackUserToCache(req.UserID, req.BlackUserID); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) ReduceBlackUserFromCache(_ context.Context, req *pbCache.ReduceBlackUserFromCacheReq) (resp *pbCache.ReduceBlackUserFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.ReduceBlackUserFromCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.ReduceBlackUserFromCache(req.UserID, req.BlackUserID); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) GetGroupMemberIDListFromCache(_ context.Context, req *pbCache.GetGroupMemberIDListFromCacheReq) (resp *pbCache.GetGroupMemberIDListFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.GetGroupMemberIDListFromCacheResp{
		CommonResp: &pbCache.CommonResp{},
	}
	userIDList, err := db.DB.GetGroupMemberIDListFromCache(req.GroupID)
	if err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "GetGroupMemberIDListFromCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	resp.UserIDList = userIDList
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) AddGroupMemberToCache(_ context.Context, req *pbCache.AddGroupMemberToCacheReq) (resp *pbCache.AddGroupMemberToCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.AddGroupMemberToCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.AddGroupMemberToCache(req.GroupID, req.UserIDList...); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "AddGroupMemberToCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

func (s *cacheServer) ReduceGroupMemberFromCache(_ context.Context, req *pbCache.ReduceGroupMemberFromCacheReq) (resp *pbCache.ReduceGroupMemberFromCacheResp, err error) {
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "req: ", req.String())
	resp = &pbCache.ReduceGroupMemberFromCacheResp{CommonResp: &pbCache.CommonResp{}}
	if err := db.DB.ReduceGroupMemberFromCache(req.GroupID, req.UserIDList...); err != nil {
		log.NewError(req.OperationID, utils.GetSelfFuncName(), "ReduceGroupMemberFromCache failed", err.Error())
		resp.CommonResp.ErrCode = constant.ErrDB.ErrCode
		resp.CommonResp.ErrMsg = constant.ErrDB.ErrMsg
		return resp, nil
	}
	log.NewInfo(req.OperationID, utils.GetSelfFuncName(), "resp: ", resp.String())
	return resp, nil
}

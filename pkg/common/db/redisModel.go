package db

import (
	"Open_IM/pkg/common/constant"
	log2 "Open_IM/pkg/common/log"
	pbCommon "Open_IM/pkg/proto/sdk_ws"
	"encoding/json"
	"github.com/garyburd/redigo/redis"
)

const (
	accountTempCode               = "ACCOUNT_TEMP_CODE"
	resetPwdTempCode              = "RESET_PWD_TEMP_CODE"
	userIncrSeq                   = "REDIS_USER_INCR_SEQ:" // user incr seq
	appleDeviceToken              = "DEVICE_TOKEN"
	userMinSeq                    = "REDIS_USER_MIN_SEQ:"
	uidPidToken                   = "UID_PID_TOKEN_STATUS:"
	conversationReceiveMessageOpt = "CON_RECV_MSG_OPT:"
	getuiToken                    = "GETUI"
	userInfoCache                 = "USER_INFO_CACHE:"
	friendRelationCache           = "FRIEND_RELATION_CACHE:"
	blackListCache                = "BLACK_LIST_CACHE:"
	groupCache                    = "GROUP_CACHE:"
)

func (d *DataBases) Exec(cmd string, key interface{}, args ...interface{}) (interface{}, error) {
	con := d.redisPool.Get()
	if err := con.Err(); err != nil {
		log2.Error("", "", "redis cmd = %v, err = %v", cmd, err)
		return nil, err
	}
	defer con.Close()

	params := make([]interface{}, 0)
	params = append(params, key)

	if len(args) > 0 {
		for _, v := range args {
			params = append(params, v)
		}
	}

	return con.Do(cmd, params...)
}
func (d *DataBases) JudgeAccountEXISTS(account string) (bool, error) {
	key := accountTempCode + account
	return redis.Bool(d.Exec("EXISTS", key))
}
func (d *DataBases) SetAccountCode(account string, code, ttl int) (err error) {
	key := accountTempCode + account
	_, err = d.Exec("SET", key, code, "ex", ttl)
	return err
}
func (d *DataBases) GetAccountCode(account string) (string, error) {
	key := accountTempCode + account
	return redis.String(d.Exec("GET", key))
}

//Perform seq auto-increment operation of user messages
func (d *DataBases) IncrUserSeq(uid string) (uint64, error) {
	key := userIncrSeq + uid
	return redis.Uint64(d.Exec("INCR", key))
}

//Get the largest Seq
func (d *DataBases) GetUserMaxSeq(uid string) (uint64, error) {
	key := userIncrSeq + uid
	return redis.Uint64(d.Exec("GET", key))
}

//Set the user's minimum seq
func (d *DataBases) SetUserMinSeq(uid string, minSeq uint32) (err error) {
	key := userMinSeq + uid
	_, err = d.Exec("SET", key, minSeq)
	return err
}

//Get the smallest Seq
func (d *DataBases) GetUserMinSeq(uid string) (uint64, error) {
	key := userMinSeq + uid
	return redis.Uint64(d.Exec("GET", key))
}

//Store Apple's device token to redis
func (d *DataBases) SetAppleDeviceToken(accountAddress, value string) (err error) {
	key := appleDeviceToken + accountAddress
	_, err = d.Exec("SET", key, value)
	return err
}

//Delete Apple device token
func (d *DataBases) DelAppleDeviceToken(accountAddress string) (err error) {
	key := appleDeviceToken + accountAddress
	_, err = d.Exec("DEL", key)
	return err
}

//Store userid and platform class to redis
func (d *DataBases) AddTokenFlag(userID string, platformID int32, token string, flag int) error {
	key := uidPidToken + userID + ":" + constant.PlatformIDToName(platformID)
	log2.NewDebug("", "add token key is ", key)
	_, err1 := d.Exec("HSet", key, token, flag)
	return err1
}

func (d *DataBases) GetTokenMapByUidPid(userID, platformID string) (map[string]int, error) {
	key := uidPidToken + userID + ":" + platformID
	log2.NewDebug("", "get token key is ", key)
	return redis.IntMap(d.Exec("HGETALL", key))
}
func (d *DataBases) SetTokenMapByUidPid(userID string, platformID int32, m map[string]int) error {
	key := uidPidToken + userID + ":" + constant.PlatformIDToName(platformID)
	_, err := d.Exec("hmset", key, redis.Args{}.Add().AddFlat(m)...)
	return err
}
func (d *DataBases) DeleteTokenByUidPid(userID string, platformID int32, fields []string) error {
	key := uidPidToken + userID + ":" + constant.PlatformIDToName(platformID)
	_, err := d.Exec("HDEL", key, redis.Args{}.Add().AddFlat(fields)...)
	return err
}

func (d *DataBases) SetSingleConversationRecvMsgOpt(userID, conversationID string, opt int32) error {
	key := conversationReceiveMessageOpt + userID
	_, err := d.Exec("HSet", key, conversationID, opt)
	return err
}

func (d *DataBases) GetSingleConversationRecvMsgOpt(userID, conversationID string) (int, error) {
	key := conversationReceiveMessageOpt + userID
	return redis.Int(d.Exec("HGet", key, conversationID))
}
func (d *DataBases) GetAllConversationMsgOpt(userID string) (map[string]int, error) {
	key := conversationReceiveMessageOpt + userID
	return redis.IntMap(d.Exec("HGETALL", key))
}
func (d *DataBases) SetMultiConversationMsgOpt(userID string, m map[string]int) error {
	key := conversationReceiveMessageOpt + userID
	_, err := d.Exec("hmset", key, redis.Args{}.Add().AddFlat(m)...)
	return err
}
func (d *DataBases) GetMultiConversationMsgOpt(userID string, conversationIDs []string) (m map[string]int, err error) {
	m = make(map[string]int)
	key := conversationReceiveMessageOpt + userID
	i, err := redis.Ints(d.Exec("hmget", key, redis.Args{}.Add().AddFlat(conversationIDs)...))
	if err != nil {
		return m, err
	}
	for k, v := range conversationIDs {
		m[v] = i[k]
	}
	return m, nil

}

func (d *DataBases) SetGetuiToken(token string, expireTime int64) error {
	_, err := d.Exec("SET", getuiToken, token, "ex", expireTime)
	return err
}

func (d *DataBases) GetGetuiToken() (string, error) {
	result, err := redis.String(d.Exec("GET", getuiToken))
	return result, err
}

func (d *DataBases) SearchContentType() {

}

func (d *DataBases) SetUserInfoToCache(userID string, m map[string]interface{}) error {
	_, err := d.Exec("hmset", userInfoCache+userID, redis.Args{}.Add().AddFlat(m)...)
	return err
}

func (d *DataBases) GetUserInfoFromCache(userID string) (*pbCommon.UserInfo, error) {
	result, err := redis.String(d.Exec("hgetall", userInfoCache+userID))
	log2.NewInfo("", result)
	if err != nil {
		return nil, err
	}
	userInfo := &pbCommon.UserInfo{}
	err = json.Unmarshal([]byte(result), userInfo)
	return userInfo, err
}

func (d *DataBases) AddFriendToCache(userID string, friendIDList ...string) error {
	var IDList []interface{}
	for _, id := range friendIDList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SADD", friendRelationCache+userID, IDList...)
	return err
}

func (d *DataBases) ReduceFriendToCache(userID string, friendIDList ...string) error {
	var IDList []interface{}
	for _, id := range friendIDList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SREM", friendRelationCache+userID, IDList...)
	return err
}

func (d *DataBases) GetFriendIDListFromCache(userID string) ([]string, error) {
	result, err := redis.Strings(d.Exec("SMEMBERS", friendRelationCache+userID))
	return result, err
}

func (d *DataBases) AddBlackUserToCache(userID string, blackList ...string) error {
	var IDList []interface{}
	for _, id := range blackList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SADD", blackListCache+userID, IDList...)
	return err
}

func (d *DataBases) ReduceBlackUserFromCache(userID string, blackList ...string) error {
	var IDList []interface{}
	for _, id := range blackList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SREM", blackListCache+userID, IDList...)
	return err
}

func (d *DataBases) GetBlackListFromCache(userID string) ([]string, error) {
	result, err := redis.Strings(d.Exec("SMEMBERS", blackListCache+userID))
	return result, err
}

func (d *DataBases) AddGroupMemberToCache(groupID string, userIDList ...string) error {
	var IDList []interface{}
	for _, id := range userIDList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SADD", groupCache+groupID, IDList...)
	return err
}

func (d *DataBases) ReduceGroupMemberFromCache(groupID string, userIDList ...string) error {
	var IDList []interface{}
	for _, id := range userIDList {
		IDList = append(IDList, id)
	}
	_, err := d.Exec("SREM", groupCache+groupID, IDList...)
	return err
}

func (d *DataBases) GetGroupMemberIDListFromCache(groupID string) ([]string, error) {
	result, err := redis.Strings(d.Exec("SMEMBERS", groupCache+groupID))
	return result, err
}

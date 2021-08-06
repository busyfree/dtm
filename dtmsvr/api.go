package dtmsvr

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yedf/dtm/common"
	"github.com/yedf/dtm/dtmcli"
)

func addRoute(engine *gin.Engine) {
	engine.POST("/api/dtmsvr/prepare", common.WrapHandler(prepare))
	engine.POST("/api/dtmsvr/submit", common.WrapHandler(submit))
	engine.POST("/api/dtmsvr/registerXaBranch", common.WrapHandler(registerXaBranch))
	engine.POST("/api/dtmsvr/registerTccBranch", common.WrapHandler(registerTccBranch))
	engine.POST("/api/dtmsvr/abort", common.WrapHandler(abort))
	engine.GET("/api/dtmsvr/query", common.WrapHandler(query))
	engine.GET("/api/dtmsvr/newGid", common.WrapHandler(newGid))
}

func newGid(c *gin.Context) (interface{}, error) {
	return M{"gid": GenGid(), "dtm_result": "SUCCESS"}, nil
}

func prepare(c *gin.Context) (interface{}, error) {
	t := TransFromContext(c)
	t.Status = "prepared"
	t.saveNew(dbGet())
	return dtmcli.ResultSuccess, nil
}

func submit(c *gin.Context) (interface{}, error) {
	db := dbGet()
	t := TransFromContext(c)
	dbt := TransFromDb(db, t.Gid)
	if dbt != nil && dbt.Status != "prepared" && dbt.Status != "submitted" {
		return M{"dtm_result": "FAILURE", "message": fmt.Sprintf("current status %s, cannot sumbmit", dbt.Status)}, nil
	}
	t.Status = "submitted"
	t.saveNew(db)
	return t.Process(db, c.Query("wait_result") == "true" || c.Query("wait_result") == "1"), nil
}

func abort(c *gin.Context) (interface{}, error) {
	db := dbGet()
	t := TransFromContext(c)
	dbt := TransFromDb(db, t.Gid)
	if t.TransType != "xa" && t.TransType != "tcc" || dbt.Status != "prepared" && dbt.Status != "aborting" {
		return M{"dtm_result": "FAILURE", "message": fmt.Sprintf("trans type: %s current status %s, cannot abort", dbt.TransType, dbt.Status)}, nil
	}
	return dbt.Process(db, c.Query("wait_result") == "true" || c.Query("wait_result") == "1"), nil
}

func registerXaBranch(c *gin.Context) (interface{}, error) {
	branch := TransBranch{}
	err := c.BindJSON(&branch)
	e2p(err)
	db := dbGet()
	dbt := TransFromDb(db, branch.Gid)
	if dbt.Status != "prepared" {
		return M{"dtm_result": "FAILURE", "message": fmt.Sprintf("current status: %s cannot register branch", dbt.Status)}, nil
	}
	branches := []TransBranch{branch, branch}
	branches[0].BranchType = "rollback"
	branches[1].BranchType = "commit"
	db.Must().Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(branches)
	e2p(err)
	global := TransGlobal{Gid: branch.Gid}
	var config = common.GetDBConfig()
	global.touch(db, config.TransCronInterval)
	return dtmcli.ResultSuccess, nil
}

func registerTccBranch(c *gin.Context) (interface{}, error) {
	data := dtmcli.MS{}
	err := c.BindJSON(&data)
	e2p(err)
	branch := TransBranch{
		Gid:      data["gid"],
		BranchID: data["branch_id"],
		Status:   data["status"],
		Data:     data["data"],
	}
	db := dbGet()
	dbt := TransFromDb(db, branch.Gid)
	if dbt.Status != "prepared" {
		return M{"dtm_result": "FAILURE", "message": fmt.Sprintf("current status: %s cannot register branch", dbt.Status)}, nil
	}

	branches := []TransBranch{branch, branch, branch}
	for i, b := range []string{"cancel", "confirm", "try"} {
		branches[i].BranchType = b
		branches[i].URL = data[b]
	}

	db.Must().Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(branches)
	e2p(err)
	global := TransGlobal{Gid: branch.Gid}
	var config = common.GetDBConfig()
	global.touch(dbGet(), config.TransCronInterval)
	return dtmcli.ResultSuccess, nil
}

func query(c *gin.Context) (interface{}, error) {
	gid := c.Query("gid")
	if gid == "" {
		return nil, errors.New("no gid specified")
	}
	trans := TransGlobal{}
	db := dbGet()
	dbr := db.Must().Where("gid", gid).First(&trans)
	if dbr.Error == gorm.ErrRecordNotFound {
		return M{"transaction": nil, "branches": [0]int{}}, nil
	}
	branches := []TransBranch{}
	db.Must().Where("gid", gid).Find(&branches)
	return M{"transaction": trans, "branches": branches}, nil
}

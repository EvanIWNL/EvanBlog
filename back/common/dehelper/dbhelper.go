// Package dbhelper TODO
package dbhelper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	mysqlErrConst "github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DefaultBatchStep 批量插入默认条数
const DefaultBatchStep = 100

var (
	// AND AND查询
	AND = "AND"
	// OR OR查询
	OR = "OR"
	// LIKE LIKE查询
	LIKE = " lk"
	// NE 不等或不在列表中
	NE = " ne"
	// NOT TODO
	NOT = "NOT"
	// IN 在列表中
	IN = " in"
)

const dbConnKeyPrefix string = "DbHelperDbConn:"

type options struct {
	dbKey string
}

// Option 连接参数
type Option func(o *options)

// DbHelper DB助手类型
type DbHelper struct {
	db   *gorm.DB
	opts *options
}

// queryOption 查询查收
type queryOption struct {
	Joins    []Join
	Select   []string
	WhereRaw []interface{}
	Group    []string
}

// QueryOption 请求参数
type QueryOption func(o *queryOption)

// Join Join关联
type Join struct {
	Type       string
	Table      string
	Conditions string
	Args       []interface{}
}

// CreateOption TODO
type CreateOption struct {
	NotIgnoreCreateAndUpdateTime bool
}

// WithNotIgnoreCreateAndUpdateTime 忽略创建时间和更新时间
func WithNotIgnoreCreateAndUpdateTime() CreateOptions {
	return func(o *CreateOption) {
		o.NotIgnoreCreateAndUpdateTime = true
	}
}

// CreateOptions 记录创建附加参数
type CreateOptions func(o *CreateOption)

// Select 查询的字段
type Select string

// WhereRaw 预定义的条件
type WhereRaw []interface{}

// WithJoins 连表查询
func WithJoins(joins []Join) QueryOption {
	return func(o *queryOption) {
		o.Joins = joins
	}
}

// WithSelect 查询字段
func WithSelect(selectColumn []string) QueryOption {
	return func(o *queryOption) {
		o.Select = selectColumn
	}
}

// WithGroup groupby
func WithGroup(group []string) QueryOption {
	return func(o *queryOption) {
		o.Group = group
	}
}

// WithWhereRaw 预定义查询条件
func WithWhereRaw(whereRaw []interface{}) QueryOption {
	return func(o *queryOption) {
		o.WhereRaw = whereRaw
	}
}

// WithDbKey 设置db key方法
func WithDbKey(dbKey string) Option {
	return func(o *options) {
		o.dbKey = dbKey
	}
}

// NewDbHelper 构造函数
func NewDbHelper(db *gorm.DB, opts ...Option) *DbHelper {
	hpOpts := &options{}
	for _, o := range opts {
		o(hpOpts)
	}
	hp := &DbHelper{
		db:   db,
		opts: hpOpts,
	}
	return hp
}

// GetDbKey 获取db key
func (hp *DbHelper) GetDbKey() string {
	dbKey := ""
	if hp.opts != nil {
		dbKey = hp.opts.dbKey
	}
	return dbKey
}

func (hp *DbHelper) withDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbConnKeyPrefix+hp.GetDbKey(), db) // set进ctx的时候，带上db的key，防止同个ctx内使用多个db的时候出现覆盖
}

// GetDb 获取db连接
func (hp *DbHelper) GetDb(ctx context.Context) *gorm.DB {
	v := ctx.Value(dbConnKeyPrefix + hp.GetDbKey()) // 优先使用ctx中的db，保证在一个事务内
	db, ok := v.(*gorm.DB)
	if ok && db != nil {
		return db.WithContext(ctx)
	}

	return hp.db.WithContext(ctx)
}

// SetDb 设置db连接
func (hp *DbHelper) SetDb(db *gorm.DB) {
	hp.db = db
}

// IsDupEntryErr 判断db错误是否为唯一键冲突
func IsDupEntryErr(dbErr error) bool {
	mysqlErr, ok := errors.Unwrap(dbErr).(*mysql.MySQLError)
	if ok && mysqlErr.Number == mysqlErrConst.ER_DUP_ENTRY {
		return true
	}

	return false
}

// AddRecord 新增记录
func (hp *DbHelper) AddRecord(ctx context.Context, table string, data interface{}) error {
	db := hp.GetDb(ctx)
	err := db.Table(table).
		Create(data).
		Scan(data).Error // 如果data中有id字段，insert id会写到id字段上
	if err != nil {
		return err
	}

	return nil
}

// AddRecordBatchStep 分批新增记录
func (hp *DbHelper) AddRecordBatchStep(ctx context.Context, table string, records interface{}, step int) error {
	db := hp.GetDb(ctx)
	err := db.Table(table).
		CreateInBatches(records, step).Error
	if err != nil {
		return err
	}

	return nil
}

// DeleteRecord 删除记录
func (hp *DbHelper) DeleteRecord(
	ctx context.Context,
	table string,
	conditions map[string]interface{},
	limit int,
) (rowsAffected int64, err error) {
	query := hp.buildQuery(ctx, table, conditions)
	if limit > 0 {
		query = query.Limit(limit)
	}
	var model interface{}
	query = query.Delete(model)
	rowsAffected, err = query.RowsAffected, query.Error
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// GetList 列表查询，res可以为单个对象，也可以为列表（对象或map列表），但都必须是指针
func (hp *DbHelper) GetList(
	ctx context.Context,
	table string,
	res interface{},
	conditions map[string]interface{},
	order string,
	limit int,
	opts ...QueryOption,
) error {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	if len(order) > 0 {
		query = query.Order(order)
	}
	err := query.Limit(limit).Find(res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// Sum 获取计数
func (hp *DbHelper) Sum(
	ctx context.Context,
	table string,
	conditions map[string]interface{},
	columns string,
	opts ...QueryOption,
) (uint64, error) {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	res := struct {
		Total uint64 `gorm:"column:total"`
	}{}
	err := query.Select("SUM(" + columns + ") as total").Take(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return res.Total, nil
}

// Take 获取单条记录
func (hp *DbHelper) Take(ctx context.Context, table string, res interface{}, conditions map[string]interface{},
	order string, opts ...QueryOption) error {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	query.Order(order)
	err := query.Take(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// First 获取单条记录
func (hp *DbHelper) First(ctx context.Context, table string, res interface{}, conditions map[string]interface{},
	opts ...QueryOption) error {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	err := query.First(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// GetListByPage 分页查询，res可以为单个对象，也可以为列表（对象或map列表），但都必须是指针
func (hp *DbHelper) GetListByPage(
	ctx context.Context,
	table string,
	res interface{},
	conditions map[string]interface{},
	order string,
	limit int,
	offset int,
	opts ...QueryOption,
) error {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	if len(order) > 0 {
		query = query.Order(order)
	}
	err := query.Limit(limit).Offset(offset).Find(res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

func (hp *DbHelper) buildQuery(
	ctx context.Context,
	table string,
	conditions map[string]interface{},
	opts ...QueryOption,
) *gorm.DB {
	queryOpts := &queryOption{}
	for _, o := range opts {
		o(queryOpts)
	}
	db := hp.GetDb(ctx)
	query := db.Table(table)
	if len(queryOpts.Joins) > 0 {
		for _, v := range queryOpts.Joins {
			if v.Table == "" && v.Conditions == "" {
				query = query.Joins(v.Type, v.Args...)
			} else {
				query = query.Joins(fmt.Sprintf("%s join %s on %s", v.Type, v.Table, v.Conditions),
					v.Args...)
			}
		}
	}
	if len(queryOpts.Select) > 0 {
		query = query.Select(strings.Join(queryOpts.Select, ","))
	}
	if len(conditions) > 0 {
		query = hp.buildCondition(ctx, query, conditions, AND)
	}
	if len(queryOpts.WhereRaw) > 0 {
		for _, v := range queryOpts.WhereRaw {
			query = query.Where(v)
		}
	}
	if len(queryOpts.Group) > 0 {
		for _, v := range queryOpts.Group {
			query = query.Group(v)
		}
	}
	return query
}

// buildCondition 组建查询条件
// 支持复杂的查询方式
//
//	eg: conditions = map[string]interface{}{
//			"k1":    "v1",
//			"k2 lk": "v2",
//			"OR": map[string]interface{}{
//				"k3": "v3",
//				"AND": map[string]interface{}{
//					"k4": "v4",
//					"k5": "v5",
//					"OR": map[string]interface{}{
//						"k8": "v8",
//						"k9": "v9",
//					},
//				},
//			},
//			"k6 ne": []string{"v66", "v67"},
//		}
//
// sql语句为:SELECT * FROM `table` WHERE
// `k1` = 'v1' AND
// `k2` like '%v2%' AND
//
//	(
//		`k3` = 'v3' OR (`k4` = 'v4' AND `k5` = 'v5' AND (`k8` = 'v8' OR `k9` = 'v9')
//	) AND
//	`k6` NOT IN ('v66','v67')
func (hp *DbHelper) buildCondition(ctx context.Context, tx *gorm.DB, conditions map[string]interface{},
	typ string) *gorm.DB {
	if typ == "" {
		typ = "AND"
	}
	for k, v := range conditions {
		var (
			query     interface{}
			args      interface{}
			queryType = typ
		)
		if len(k) >= 4 && k[len(k)-3:] == LIKE {
			query = strings.Replace(k, LIKE, " like ?", 1)
			args = "%" + v.(string) + "%"
		} else if len(k) >= 4 && k[len(k)-3:] == NE {
			query = strings.Replace(k, NE, "", 1)
			queryType = NOT
			args = v
		} else if len(k) >= 4 && k[len(k)-3:] == IN {
			query = strings.Replace(k, IN, "", 1)
			queryType = IN
			args = v
		} else if (isAND(k) || isOR(k)) && reflect.TypeOf(v).Kind() == reflect.Map {
			query = hp.buildCondition(ctx, hp.GetDb(ctx), v.(map[string]interface{}), k)
		} else {
			query = k
			args = v
		}
		tx = hp.andOr(tx, queryType, query, args)
	}
	return tx
}

func (hp *DbHelper) andOr(tx *gorm.DB, typ string, query interface{}, args interface{}) *gorm.DB {
	if typ == "" || isAND(typ) {
		if args != nil {
			tx = tx.Where(query, args)
		} else {
			tx = tx.Where(query)
		}
	} else if typ == NOT {
		if args != nil {
			tx = tx.Not(query, args)
		} else {
			tx = tx.Not(query)
		}
	} else {
		if args != nil {
			tx = tx.Or(query, args)
		} else {
			tx = tx.Or(query)
		}

	}
	return tx
}

// Count 根据条件计算总记录数
func (hp *DbHelper) Count(
	ctx context.Context,
	table string,
	conditions map[string]interface{},
	opts ...QueryOption,
) (total int64, err error) {
	query := hp.buildQuery(ctx, table, conditions, opts...)
	err = query.Count(&total).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return total, nil
}

// UpdateRecord 更新记录，data支持map和struct
func (hp *DbHelper) UpdateRecord(
	ctx context.Context,
	table string,
	data interface{},
	conditions map[string]interface{},
	limit int,
	// zeroField ...string,
) (rowsAffected int64, err error) {
	db := hp.GetDb(ctx)
	query := db.Table(table)
	for key, values := range conditions {
		query = query.Where(key, values)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	//if len(zeroField) > 0 {
	//	_ = db.Use(zerofield.NewPlugin())
	//	query = query.Scopes(zerofield.UpdateScopes(zeroField...))
	//}
	query = query.Updates(data)
	rowsAffected, err = query.RowsAffected, query.Error
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

// InsertUpdate insert on duplicate update
func (hp *DbHelper) InsertUpdate(
	ctx context.Context,
	table string,
	data interface{},
	dupUpdateQuery map[string]interface{},
) error {
	db := hp.GetDb(ctx)
	err := db.Table(table).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.Assignments(dupUpdateQuery),
		}).Create(data).Error
	if err != nil {
		return err
	}

	return nil
}

// InsertIgnore insert ignore
func (hp *DbHelper) InsertIgnore(ctx context.Context, table string, data interface{}) error {
	db := hp.GetDb(ctx)
	err := db.Table(table).
		Clauses(clause.Insert{
			Modifier: "IGNORE",
		}).Create(data).Error
	if err != nil {
		return err
	}

	return nil
}

// TransBegin 开启事务，db连接保存在返回的newCtx上，需要使用newCtx进行commit/rollback
func (hp *DbHelper) TransBegin(ctx context.Context) (newCtx context.Context, err error) {
	opts := make([]*sql.TxOptions, 0)
	tx := hp.db.Begin(opts...)
	err = tx.Error
	if err != nil {
		return nil, err
	}

	newCtx = hp.withDB(ctx, tx)
	return newCtx, nil
}

// TransCommit 提交事务
func (hp *DbHelper) TransCommit(ctx context.Context) (newCtx context.Context, err error) {
	tx := hp.GetDb(ctx).Commit()
	newCtx = hp.withDB(ctx, nil)
	err = tx.Error
	if err != nil {
		return nil, err
	}

	return newCtx, nil
}

// TransRollback 回滚事务
func (hp *DbHelper) TransRollback(ctx context.Context) (newCtx context.Context, err error) {
	tx := hp.GetDb(ctx).Rollback()
	newCtx = hp.withDB(ctx, nil)
	err = tx.Error
	if err != nil {
		return nil, err
	}

	return newCtx, nil
}

// RawQuery 原生sql查询，res可以为单个对象，也可以为列表
func (hp *DbHelper) RawQuery(ctx context.Context, sql string, params []interface{}, res interface{}) error {
	db := hp.GetDb(ctx)
	err := db.Raw(sql, params...).Scan(res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// ExecSql 执行原生sql
func (hp *DbHelper) ExecSql(ctx context.Context, sql string, params []interface{}) error {
	db := hp.GetDb(ctx)
	err := db.Exec(sql, params...).Error
	if err != nil {
		return err
	}

	return nil
}

// SoftDeleteRecordById 根据id软删除记录（要求数据表主键名为id，软删除标志位为deleted）
func (hp *DbHelper) SoftDeleteRecordById(ctx context.Context, table string, id uint) error {
	conditions := map[string]interface{}{
		"id": id,
	}
	delData := map[string]interface{}{
		"deleted": id,
	}
	limit := 1
	_, err := hp.UpdateRecord(ctx, table, &delData, conditions, limit)
	return err
}

// GetNotDeletedList 获取未被软删除的数据列表
func (hp *DbHelper) GetNotDeletedList(
	ctx context.Context,
	table string,
	res interface{},
	conditions map[string]interface{},
) error {
	db := hp.GetDb(ctx)
	query := db.Table(table).Where("deleted", 0)
	for key, values := range conditions {
		query = query.Where(key, values)
	}
	err := query.Find(res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}

	return nil
}

// OmitCreatTime 创建忽略时间
func (hp *DbHelper) OmitCreatTime(db *gorm.DB) *gorm.DB {
	return db.Omit("updateTime", "createTime", "updated_at", "created_at")
}

func isOR(key string) bool {
	if len(key) < 2 {
		return false
	}
	return key[:2] == OR
}

func isAND(key string) bool {
	if len(key) < 3 {
		return false
	}
	return key[:3] == AND
}

func (m *default{{.upperStartCamelObject}}Model) Update(ctx context.Context, session sqlx.Session, {{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) (sql.Result, error) {
	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}})
        if err!=nil{
            return nil,err
        }
     {{end}}{{.keys}}
	return m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
	query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}})
	}
	return conn.ExecCtx(ctx, query, {{.expressionValues}})
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}}", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}})
	}
	return m.conn.ExecCtx(ctx, query, {{.expressionValues}}){{end}}
}

func (m *default{{.upperStartCamelObject}}Model) UpdateWithVersion(ctx context.Context,session sqlx.Session,{{if .containsIndexCache}}newData{{else}}data{{end}} *{{.upperStartCamelObject}}) error {

    {{if .containsIndexCache}}
     oldVersion := newData.Version
     newData.Version += 1
    {{else}}
    oldVersion := data.Version
    data.Version += 1
    {{end}}

	var sqlResult sql.Result
	var err error

	{{if .withCache}}{{if .containsIndexCache}}data, err:=m.FindOne(ctx, newData.{{.upperStartCamelPrimaryKey}})
            if err!=nil{
                return err
            }
    {{end}}{{.keys}}
	sqlResult,err =  m.ExecCtx(ctx,func(ctx context.Context,conn sqlx.SqlConn) (result sql.Result, err error) {
	query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ? ", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		return session.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}
	return conn.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set %s where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ? ", m.table, {{.lowerStartCamelObject}}RowsWithPlaceHolder)
	if session != nil{
		sqlResult,err  =  session.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}else{
		sqlResult,err  =  m.conn.ExecCtx(ctx,query, {{.expressionValues}},oldVersion)
	}
	{{end}}
	if err != nil {
		return err
	}
	updateCount , err := sqlResult.RowsAffected()
	if err != nil{
		return err
	}
	if updateCount == 0 {
		return ErrNoRowsUpdate
	}

	return nil
}

func (m *default{{.upperStartCamelObject}}Model) DeleteSoft(ctx context.Context,session sqlx.Session,data *{{.upperStartCamelObject}}) error {
	data.DelState = globalkey.DelStateYes
	data.DeleteTime = sql.NullTime{
		Time: time.Now(),
		Valid: true,
	}
	data.Version += 1

	var sqlResult sql.Result
	var err error

	{{if .withCache}}{{.keys}}
	sqlResult, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (result sql.Result, err error) {
		query := fmt.Sprintf("update %s set del_state = ?, delete_time = ?, version = ? where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ?", m.table)
		if session != nil {
			return session.ExecCtx(ctx, query, globalkey.DelStateYes, data.DeleteTime, data.Version, data.{{.upperStartCamelPrimaryKey}}, data.Version-1)
		}
		return conn.ExecCtx(ctx, query, globalkey.DelStateYes, data.DeleteTime, data.Version, data.{{.upperStartCamelPrimaryKey}}, data.Version-1)
	}, {{.keyValues}}){{else}}query := fmt.Sprintf("update %s set del_state = ?, delete_time = ?, version = ? where {{.originalPrimaryKey}} = {{if .postgreSql}}$1{{else}}?{{end}} and version = ?", m.table)
	if session != nil {
		sqlResult, err = session.ExecCtx(ctx, query, globalkey.DelStateYes, data.DeleteTime, data.Version, data.{{.upperStartCamelPrimaryKey}}, data.Version-1)
	} else {
		sqlResult, err = m.conn.ExecCtx(ctx, query, globalkey.DelStateYes, data.DeleteTime, data.Version, data.{{.upperStartCamelPrimaryKey}}, data.Version-1)
	}{{end}}
	if err != nil {
		return errors.Wrapf(errors.New("delete soft failed"), "{{.upperStartCamelObject}}Model delete err : %+v", err)
	}
	updateCount, err := sqlResult.RowsAffected()
	if err != nil {
		return err
	}
	if updateCount == 0 {
		return ErrNoRowsUpdate
	}
	return nil
}
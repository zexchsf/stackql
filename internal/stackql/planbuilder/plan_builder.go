package planbuilder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"

	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	// only string "false" will disable
	PlanCacheEnabled string = "true"
)

type PlanBuilderInput interface {
	GetAliasedTables() parserutil.TableAliasMap
	GetAuth() (*sqlparser.Auth, bool)
	GetAuthRevoke() (*sqlparser.AuthRevoke, bool)
	GetAssignedAliasedColumns() map[sqlparser.TableName]sqlparser.TableExpr
	GetColRefs() parserutil.ColTableMap
	GetDelete() (*sqlparser.Delete, bool)
	GetDescribeTable() (*sqlparser.DescribeTable, bool)
	GetExec() (*sqlparser.Exec, bool)
	GetHandlerCtx() handler.HandlerContext
	GetInsert() (*sqlparser.Insert, bool)
	GetNativeQuery() (*sqlparser.NativeQuery, bool)
	GetPlaceholderParams() parserutil.ParameterMap
	GetPurge() (*sqlparser.Purge, bool)
	GetRawQuery() string
	GetRegistry() (*sqlparser.Registry, bool)
	GetSelect() (*sqlparser.Select, bool)
	GetShow() (*sqlparser.Show, bool)
	GetSleep() (*sqlparser.Sleep, bool)
	GetStatement() sqlparser.SQLNode
	GetTableExprs() sqlparser.TableExprs
	GetTxnCtrlCtrs() internaldto.TxnControlCounters
	GetUnion() (*sqlparser.Union, bool)
	GetUpdate() (*sqlparser.Update, bool)
	GetUse() (*sqlparser.Use, bool)
}

func isPlanCacheEnabled() bool {
	return strings.ToLower(PlanCacheEnabled) != "false"
}

type StandardPlanBuilderInput struct {
	handlerCtx             handler.HandlerContext
	stmt                   sqlparser.SQLNode
	colRefs                parserutil.ColTableMap
	aliasedTables          parserutil.TableAliasMap
	assignedAliasedColumns parserutil.TableExprMap
	tables                 sqlparser.TableExprs
	paramsPlaceheld        parserutil.ParameterMap
	tcc                    internaldto.TxnControlCounters
}

func NewPlanBuilderInput(
	handlerCtx handler.HandlerContext,
	stmt sqlparser.SQLNode,
	tables sqlparser.TableExprs,
	assignedAliasedColumns parserutil.TableExprMap,
	aliasedTables parserutil.TableAliasMap,
	colRefs parserutil.ColTableMap,
	paramsPlaceheld parserutil.ParameterMap,
	tcc internaldto.TxnControlCounters,
) (PlanBuilderInput, error) {
	rv := &StandardPlanBuilderInput{
		handlerCtx:             handlerCtx,
		stmt:                   stmt,
		tables:                 tables,
		aliasedTables:          aliasedTables,
		assignedAliasedColumns: assignedAliasedColumns,
		colRefs:                colRefs,
		paramsPlaceheld:        paramsPlaceheld,
		tcc:                    tcc,
	}
	if handlerCtx == nil {
		return nil, fmt.Errorf("plan builder input invariant violation: nil handler context")
	}
	if rv.assignedAliasedColumns == nil {
		rv.assignedAliasedColumns = make(map[sqlparser.TableName]sqlparser.TableExpr)
	}
	return rv, nil
}

func (pbi *StandardPlanBuilderInput) GetRawQuery() string {
	return pbi.handlerCtx.GetRawQuery()
}

func (pbi *StandardPlanBuilderInput) GetStatement() sqlparser.SQLNode {
	return pbi.stmt
}

func (pbi *StandardPlanBuilderInput) GetTxnCtrlCtrs() internaldto.TxnControlCounters {
	return pbi.tcc
}

func (pbi *StandardPlanBuilderInput) GetPlaceholderParams() parserutil.ParameterMap {
	return pbi.paramsPlaceheld
}

func (pbi *StandardPlanBuilderInput) GetAssignedAliasedColumns() map[sqlparser.TableName]sqlparser.TableExpr {
	return pbi.assignedAliasedColumns
}

func (pbi *StandardPlanBuilderInput) GetAliasedTables() parserutil.TableAliasMap {
	return pbi.aliasedTables
}

func (pbi *StandardPlanBuilderInput) GetColRefs() parserutil.ColTableMap {
	return pbi.colRefs
}

func (pbi *StandardPlanBuilderInput) GetTableExprs() sqlparser.TableExprs {
	return pbi.tables
}

func (pbi *StandardPlanBuilderInput) GetAuth() (*sqlparser.Auth, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Auth)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetAuthRevoke() (*sqlparser.AuthRevoke, bool) {
	rv, ok := pbi.stmt.(*sqlparser.AuthRevoke)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetDelete() (*sqlparser.Delete, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Delete)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetDescribeTable() (*sqlparser.DescribeTable, bool) {
	rv, ok := pbi.stmt.(*sqlparser.DescribeTable)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetExec() (*sqlparser.Exec, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Exec)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetInsert() (*sqlparser.Insert, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Insert)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetRegistry() (*sqlparser.Registry, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Registry)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetPurge() (*sqlparser.Purge, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Purge)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetNativeQuery() (*sqlparser.NativeQuery, bool) {
	rv, ok := pbi.stmt.(*sqlparser.NativeQuery)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetSelect() (*sqlparser.Select, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Select)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetShow() (*sqlparser.Show, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Show)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetSleep() (*sqlparser.Sleep, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Sleep)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUnion() (*sqlparser.Union, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Union)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUse() (*sqlparser.Use, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Use)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUpdate() (*sqlparser.Update, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Update)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetHandlerCtx() handler.HandlerContext {
	return pbi.handlerCtx
}

type planGraphBuilder struct {
	planGraph *primitivegraph.PrimitiveGraph
}

func newPlanGraphBuilder(concurrencyLimit int) *planGraphBuilder {
	return &planGraphBuilder{
		planGraph: primitivegraph.NewPrimitiveGraph(concurrencyLimit),
	}
}

func (pgb *planGraphBuilder) createInstructionFor(pbi PlanBuilderInput) error {
	stmt := pbi.GetStatement()
	switch stmt := stmt.(type) {
	case *sqlparser.Auth:
		return pgb.handleAuth(pbi)
	case *sqlparser.AuthRevoke:
		return pgb.handleAuthRevoke(pbi)
	case *sqlparser.Begin:
		return pgb.nop(pbi)
	case *sqlparser.Commit:
		return pgb.nop(pbi)
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return iqlerror.GetStatementNotSupportedError("DDL")
	case *sqlparser.Delete:
		return pgb.handleDelete(pbi)
	case *sqlparser.DescribeTable:
		return pgb.handleDescribe(pbi)
	case *sqlparser.Exec:
		return pgb.handleExec(pbi)
	case *sqlparser.Explain:
		return iqlerror.GetStatementNotSupportedError("EXPLAIN")
	case *sqlparser.Insert:
		return pgb.handleInsert(pbi)
	case *sqlparser.NativeQuery:
		return pgb.handleNativeQuery(pbi)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Purge:
		return pgb.handlePurge(pbi)
	case *sqlparser.Registry:
		return pgb.handleRegistry(pbi)
	case *sqlparser.Rollback:
		return pgb.nop(pbi)
	case *sqlparser.Savepoint:
		return pgb.nop(pbi)
	case *sqlparser.Select:
		_, _, err := pgb.handleSelect(pbi)
		return err
	case *sqlparser.Set:
		return pgb.nop(pbi)
	case *sqlparser.SetTransaction:
		return pgb.nop(pbi)
	case *sqlparser.Show:
		return pgb.handleShow(pbi)
	case *sqlparser.Sleep:
		return pgb.handleSleep(pbi)
	case *sqlparser.SRollback:
		return pgb.nop(pbi)
	case *sqlparser.Release:
		return pgb.nop(pbi)
	case *sqlparser.Union:
		_, _, err := pgb.handleUnion(pbi)
		return err
	case *sqlparser.Update:
		return pgb.handleUpdate(pbi)
	case *sqlparser.Use:
		return pgb.handleUse(pbi)
	}
	return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("BUG: unexpected statement type: %T", stmt))
}

func (pgb *planGraphBuilder) nop(pbi PlanBuilderInput) error {
	primitiveGenerator := newRootPrimitiveGenerator(nil, pbi.GetHandlerCtx(), pgb.planGraph)
	err := primitiveGenerator.analyzeNop(pbi)
	return err
}

func (pgb *planGraphBuilder) pgInternal(pbi PlanBuilderInput) error {
	primitiveGenerator := newRootPrimitiveGenerator(nil, pbi.GetHandlerCtx(), pgb.planGraph)
	err := primitiveGenerator.analyzePGInternal(pbi)
	if err != nil {
		return err
	}
	builder := primitiveGenerator.PrimitiveComposer.GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil pg internal builder")
	}
	err = builder.Build()
	return err
}

func (pgb *planGraphBuilder) handleAuth(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetAuth()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Auth", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	err = primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Debugln(fmt.Sprintf("err = %s", err.Error()))
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			authType := strings.ToLower(node.Type)
			if node.KeyFilePath != "" {
				authCtx.KeyFilePath = node.KeyFilePath
			}
			if node.KeyEnvVar != "" {
				authCtx.KeyEnvVar = node.KeyEnvVar
			}
			_, err := prov.Auth(authCtx, authType, true)
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleAuthRevoke(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	stmt := pbi.GetStatement()
	node, ok := stmt.(*sqlparser.AuthRevoke)
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required AuthRevoke", stmt)
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		return err
	}
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, prov.AuthRevoke(authCtx))
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleDescribe(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDescribeTable()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DescribeTable", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		return err
	}
	md, err := primitiveGenerator.PrimitiveComposer.GetTable(node)
	if err != nil {
		return err
	}
	prov, err := md.GetProvider()
	if err != nil {
		return err
	}
	var extended bool = strings.TrimSpace(strings.ToUpper(node.Extended)) == "EXTENDED"
	var full bool = strings.TrimSpace(strings.ToUpper(node.Full)) == "FULL"
	pr := primitive.NewMetaDataPrimitive(
		prov,
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitiveGenerator.describeInstructionExecutor(handlerCtx, md, extended, full)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleSelect(pbi PlanBuilderInput) (*primitivegraph.PrimitiveNode, *primitivegraph.PrimitiveNode, error) {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.analyzeStatement(pbi)
		if err != nil {
			logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
			return nil, nil, err
		}
		builder := primitiveGenerator.PrimitiveComposer.GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		_, isRawNativeSelect := builder.(*primitivebuilder.RawNativeSelect)
		_, isRawNativeExec := builder.(*primitivebuilder.RawNativeExec)
		isLocallyExecutable := !isNativeSelect && !isRawNativeSelect && !isRawNativeExec
		// check tables only if not native
		if isLocallyExecutable {
			for _, val := range primitiveGenerator.PrimitiveComposer.GetTables() {
				isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
			}
		}
		if isLocallyExecutable {
			pr, err := primitiveGenerator.localSelectExecutor(handlerCtx, node, util.DefaultRowSort)
			if err != nil {
				return nil, nil, err
			}
			rv := pgb.planGraph.CreatePrimitiveNode(pr)
			return &rv, &rv, nil
		}
		if primitiveGenerator.PrimitiveComposer.GetBuilder() == nil {
			return nil, nil, fmt.Errorf("builder not created for select, cannot proceed")
		}
		err = builder.Build()
		if err != nil {
			return nil, nil, err
		}
		root := builder.GetRoot()
		tail := builder.GetTail()
		return &root, &tail, nil
	}
	pr := primitive.NewLocalPrimitive(nil)
	rv := pgb.planGraph.CreatePrimitiveNode(pr)
	return &rv, &rv, nil
}

func (pgb *planGraphBuilder) handleUnion(pbi PlanBuilderInput) (*primitivegraph.PrimitiveNode, *primitivegraph.PrimitiveNode, error) {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUnion()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
		return nil, nil, err
	}
	isLocallyExecutable := true
	for _, val := range primitiveGenerator.PrimitiveComposer.GetTables() {
		isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
	}
	if primitiveGenerator.PrimitiveComposer.GetBuilder() == nil {
		return nil, nil, fmt.Errorf("builder not created for union, cannot proceed")
	}
	builder := primitiveGenerator.PrimitiveComposer.GetBuilder()
	err = builder.Build()
	if err != nil {
		return nil, nil, err
	}
	root := builder.GetRoot()
	tail := builder.GetTail()
	return &root, &tail, nil
}

func (pgb *planGraphBuilder) handleDelete(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDelete()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.analyzeStatement(pbi)
		if err != nil {
			return err
		}
		pr, err := primitiveGenerator.deleteExecutor(handlerCtx, node)
		if err != nil {
			return err
		}
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
}

func (pgb *planGraphBuilder) handleRegistry(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Registry", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeRegistry(pbi)
	if err != nil {
		return err
	}
	reg, err := handler.GetRegistry(handlerCtx.GetRuntimeContext())
	if err != nil {
		return err
	}
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			switch at := strings.ToLower(node.ActionType); at {
			case "pull":
				err := reg.PullAndPersistProviderArchive(node.ProviderId, node.ProviderVersion)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, nil, nil, nil, nil, &internaldto.BackendMessages{WorkingMessages: []string{fmt.Sprintf("%s provider, version '%s' successfully installed", node.ProviderId, node.ProviderVersion)}}, nil))
			case "list":
				var colz []string
				var provz map[string]openapistackql.ProviderDescription
				keys := make(map[string]map[string]interface{})
				if node.ProviderId == "" {
					provz, err = reg.ListAllAvailableProviders()
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "version"}
					var dks []string
					for k, _ := range provz {
						dks = append(dks, k)
					}
					sort.Strings(dks)
					for i, k := range dks {
						v := provz[k]
						for _, ver := range v.Versions {
							keys[strconv.Itoa(i)] = map[string]interface{}{
								"provider": k,
								"version":  ver,
							}
						}
					}
				} else {
					provz, err = reg.ListAllProviderVersions(node.ProviderId)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "versions"}
					i := 0
					for k, v := range provz {
						keys[strconv.Itoa(i)] = map[string]interface{}{
							"provider": k,
							"versions": strings.Join(v.Versions, ", "),
						}
						i++
					}
				}
				return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, keys, colz, nil, nil, nil, nil))
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("registry action '%s' no supported", at))
			}
		},
	)
	pgb.planGraph.CreatePrimitiveNode(pr)

	return nil
}

func (pgb *planGraphBuilder) handlePurge(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetPurge()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	// primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	pr := primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			if node.IsGlobal {
				err := handlerCtx.GetGarbageCollector().Purge()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				return util.PrepareResultSet(
					internaldto.NewPrepareResultSetPlusRawDTO(
						nil,
						map[string]map[string]interface{}{"0": {"message": "purge 'GLOBAL' completed"}},
						[]string{"message"},
						nil,
						nil,
						&internaldto.BackendMessages{
							WorkingMessages: []string{fmt.Sprintf("Global PURGE successfully completed")}},
						nil,
					),
				)
			}
			targetStr := strings.ToLower(node.Target.GetRawVal())
			switch targetStr {
			case "cache":
				err := handlerCtx.GetGarbageCollector().PurgeCache()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "conservative":
				err := handlerCtx.GetGarbageCollector().Collect()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "control_tables":
				err := handlerCtx.GetGarbageCollector().PurgeControlTables()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "ephemeral":
				err := handlerCtx.GetGarbageCollector().PurgeEphemeral()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("purge target '%s' not supported", targetStr))
			}
			// This happens in all cases, provided the ourge is successful.
			handlerCtx.GetLRUCache().Clear()
			purgeMsg := fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					map[string]map[string]interface{}{"0": {"message": purgeMsg}},
					[]string{"message"},
					nil,
					nil,
					nil,
					// &internaldto.BackendMessages{
					// 	WorkingMessages: []string{fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)}},
					nil,
				),
			)
		},
	)
	pgb.planGraph.CreatePrimitiveNode(pr)

	return nil
}

func (pgb *planGraphBuilder) handleNativeQuery(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetNativeQuery()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	rns := primitivebuilder.NewRawNativeSelect(pgb.planGraph, handlerCtx, pbi.GetTxnCtrlCtrs(), node.QueryString)

	err := rns.Build()

	if err != nil {
		return err
	}

	return nil
}

func (pgb *planGraphBuilder) handleInsert(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetInsert()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.analyzeInsert(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractInsertValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode *primitivegraph.PrimitiveNode
		if nonValCols > 0 {
			switch rowsNode := node.Rows.(type) {
			case *sqlparser.Select:
				selPbi, err := NewPlanBuilderInput(pbi.GetHandlerCtx(), rowsNode, pbi.GetTableExprs(), pbi.GetAssignedAliasedColumns(), pbi.GetAliasedTables(), pbi.GetColRefs(), pbi.GetPlaceholderParams(), pbi.GetTxnCtrlCtrs())
				if err != nil {
					return err
				}
				_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("insert with rows of type '%T' not currently supported", rowsNode)
			}
		} else {
			selectPrimitive, err = primitiveGenerator.insertableValsExecutor(handlerCtx, insertValOnlyRows)
			if err != nil {
				return err
			}
			sn := pgb.planGraph.CreatePrimitiveNode(selectPrimitive)
			selectPrimitiveNode = &sn
		}
		pr, err := primitiveGenerator.insertExecutor(handlerCtx, node, util.DefaultRowSort)
		if err != nil {
			return err
		}
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		pr.SetInputAlias("", selectPrimitiveNode.ID())
		prNode := pgb.planGraph.CreatePrimitiveNode(pr)
		pgb.planGraph.NewDependency(*selectPrimitiveNode, prNode, 1.0)
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
	return nil
}

func (pgb *planGraphBuilder) handleUpdate(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUpdate()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.analyzeUpdate(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractUpdateValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode *primitivegraph.PrimitiveNode
		if len(nonValCols) > 0 {
			// TODO: support dynamic content
			return fmt.Errorf("update does not currently support dynamic content")
		} else {
			selectPrimitive, err = primitiveGenerator.updateableValsExecutor(handlerCtx, insertValOnlyRows)
			if err != nil {
				return err
			}
			sn := pgb.planGraph.CreatePrimitiveNode(selectPrimitive)
			selectPrimitiveNode = &sn
		}
		pr, err := primitiveGenerator.insertExecutor(handlerCtx, node, util.DefaultRowSort)
		if err != nil {
			return err
		}
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		pr.SetInputAlias("", selectPrimitiveNode.ID())
		prNode := pgb.planGraph.CreatePrimitiveNode(pr)
		pgb.planGraph.NewDependency(*selectPrimitiveNode, prNode, 1.0)
		return nil
	} else {
		pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
		pgb.planGraph.CreatePrimitiveNode(pr)
		return nil
	}
}

func (pgb *planGraphBuilder) handleExec(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetExec()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Exec", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutApiCalls {
		primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
		err := primitiveGenerator.analyzeStatement(pbi)
		if err != nil {
			return err
		}
		_, err = primitiveGenerator.execExecutor(handlerCtx, node)
		if err != nil {
			return err
		}
		return nil
	}
	pr := primitive.NewHTTPRestPrimitive(nil, nil, nil, nil)
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleShow(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Show", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		return err
	}
	nodeTypeUpper := strings.ToUpper(node.Type)
	switch nodeTypeUpper {
	case "TRANSACTION_ISOLATION_LEVEL":
		builder := primitiveGenerator.PrimitiveComposer.GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		if isNativeSelect {
			err := builder.Build()
			return err
		}
		return fmt.Errorf("improper usage of 'show transaction isolation level'")
	}
	pr := primitive.NewMetaDataPrimitive(
		primitiveGenerator.PrimitiveComposer.GetProvider(),
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitiveGenerator.showInstructionExecutor(node, handlerCtx)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *planGraphBuilder) handleSleep(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSleep()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Sleep", pbi.GetStatement())
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		return err
	}
	return nil
}

func (pgb *planGraphBuilder) handleUse(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUse()
	if !ok {
		return fmt.Errorf("node type '%T' is incorrect; expected *Use", node)
	}
	primitiveGenerator := newRootPrimitiveGenerator(node, handlerCtx, pgb.planGraph)
	err := primitiveGenerator.analyzeStatement(pbi)
	if err != nil {
		return err
	}
	pr := primitive.NewMetaDataPrimitive(
		primitiveGenerator.PrimitiveComposer.GetProvider(),
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			handlerCtx.SetCurrentProvider(node.DBName.GetRawVal())
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
		})
	pgb.planGraph.CreatePrimitiveNode(pr)
	return nil
}

func createErroneousPlan(handlerCtx handler.HandlerContext, qPlan *plan.Plan, rowSort func(map[string]map[string]interface{}) []string, err error) (*plan.Plan, error) {
	qPlan.Instructions = primitive.NewLocalPrimitive(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		return util.PrepareResultSet(
			internaldto.PrepareResultSetDTO{
				OutputBody:  nil,
				Msg:         nil,
				RowMap:      nil,
				ColumnOrder: nil,
				RowSort:     rowSort,
				Err:         err,
			},
		)
	})
	return qPlan, err
}

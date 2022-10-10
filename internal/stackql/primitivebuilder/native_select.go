package primitivebuilder

import (
	"fmt"
	"io"
	"strconv"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/nativedb"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type NativeSelect struct {
	graph       *primitivegraph.PrimitiveGraph
	handlerCtx  *handler.HandlerContext
	drmCfg      drm.DRMConfig
	selectQuery nativedb.Select
	root        primitivegraph.PrimitiveNode
}

func NewNativeSelect(graph *primitivegraph.PrimitiveGraph, handlerCtx *handler.HandlerContext, selectQuery nativedb.Select) Builder {
	return &NativeSelect{
		graph:       graph,
		handlerCtx:  handlerCtx,
		drmCfg:      handlerCtx.DrmConfig,
		selectQuery: selectQuery,
	}
}

func (ss *NativeSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *NativeSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *NativeSelect) Build() error {

	selectEx := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {

		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running empty select with columns: %v", ss.selectQuery))

		var colz []string
		for _, col := range ss.selectQuery.GetColumns() {
			colz = append(colz, col.GetName())
		}
		rowStream := ss.selectQuery.GetRows()
		rowMap := make(map[string]map[string]interface{})
		if rowStream != nil {
			i := 0
			for {
				rows, err := rowStream.Read()
				if err != nil && err != io.EOF {
					return dto.NewErroneousExecutorOutput(err)
				}
				for _, row := range rows {
					rowMap[strconv.Itoa(i)] = row
					i++
				}
				if err == io.EOF {
					break
				}
			}
		}
		rv := util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, rowMap, colz, nil, nil, nil))
		if len(rowMap) > 0 {
			return rv
		}
		return util.EmptyProtectResultSet(
			rv,
			colz,
		)
	}
	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
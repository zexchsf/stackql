package relationaldto

import (
	"github.com/stackql/stackql/internal/stackql/dto"
)

var (
	_ RelationalTable = &standardRelationalTable{}
	_ RelationalTable = &standardRelationalView{}
)

type RelationalTable interface {
	GetAlias() string
	GetBaseName() string
	GetColumns() []RelationalColumn
	GetName() (string, error)
	IsView() bool
	PushBackColumn(RelationalColumn)
	WithAlias(alias string) RelationalTable
}

func NewRelationalTable(hIDs dto.HeirarchyIdentifiers, discoveryID int, name, baseName string) RelationalTable {
	return &standardRelationalTable{
		hIDs:        hIDs,
		name:        name,
		baseName:    baseName,
		discoveryID: discoveryID,
	}
}

func NewRelationalView(hIDs dto.HeirarchyIdentifiers, discoveryID int, name, baseName string) RelationalTable {
	return &standardRelationalView{
		hIDs:        hIDs,
		name:        name,
		baseName:    baseName,
		discoveryID: discoveryID,
	}
}

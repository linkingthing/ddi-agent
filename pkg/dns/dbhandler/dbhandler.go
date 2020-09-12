package dbhandler

import (
	"fmt"
	"reflect"

	restdb "github.com/zdnscloud/gorest/db"
)

func ListWithTx(resources interface{}, tx restdb.Transaction) error {
	if err := tx.Fill(map[string]interface{}{"orderby": "create_time"}, resources); err != nil {
		return fmt.Errorf("ListWithTx:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func ListByConditionWithTx(resources interface{}, cond map[string]interface{}, tx restdb.Transaction) error {
	if err := tx.Fill(cond, resources); err != nil {
		return fmt.Errorf("ListByCondition:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func GetWithTx(ID string, out interface{}, tx restdb.Transaction) (interface{}, error) {
	if err := tx.Fill(map[string]interface{}{restdb.IDField: ID}, out); err != nil {
		return nil, err
	}

	sliceVal := reflect.ValueOf(out).Elem()
	if sliceVal.Len() == 1 {
		return sliceVal.Index(0).Interface(), nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}

func ExistWithTx(table restdb.ResourceType, ID string, tx restdb.Transaction) (bool, error) {
	if exist, err := tx.Exists(table, map[string]interface{}{restdb.IDField: ID}); err != nil {
		return false, fmt.Errorf("Exist id:%s failed:%s ", ID, err.Error())
	} else {
		return exist, nil
	}
}

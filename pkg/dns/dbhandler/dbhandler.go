package dbhandler

import (
	"fmt"
	"reflect"

	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/db"
)

func Insert(res restresource.Resource) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(res); err != nil {
			return fmt.Errorf("Insert:%+v failed:%s ", res, err.Error())
		}
		return nil
	})
}

func Delete(ID string, table restdb.ResourceType) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if c, err := tx.Delete(table, map[string]interface{}{restdb.IDField: ID}); err != nil {
		return err
	} else if c == 0 {
		return nil
	}
	tx.Commit()

	return nil
}

func List(resources interface{}) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(map[string]interface{}{"orderby": "create_time"}, resources)
	}); err != nil {
		return fmt.Errorf("list:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func ListWithTx(resources interface{}, tx restdb.Transaction) error {
	if err := tx.Fill(map[string]interface{}{"orderby": "create_time"}, resources); err != nil {
		return fmt.Errorf("ListWithTx:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func ListByCondition(resources interface{}, cond map[string]interface{}) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return tx.Fill(cond, resources)
	}); err != nil {
		return fmt.Errorf("ListByCondition:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func ListByConditionWithTx(resources interface{}, cond map[string]interface{}, tx restdb.Transaction) error {
	if err := tx.Fill(cond, resources); err != nil {
		return fmt.Errorf("ListByCondition:%+v failed:%s", resources, err.Error())
	}

	return nil
}

func Get(ID string, inRes interface{}) (restresource.Resource, error) {
	outRes, err := restdb.GetResourceWithID(db.GetDB(), ID, inRes)
	if err != nil {
		return nil, fmt.Errorf("Get:%s failed:%s", ID, err.Error())
	}

	return outRes.(restresource.Resource), nil
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

func Exist(table restdb.ResourceType, ID string) (bool, error) {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if exist, err := tx.Exists(table, map[string]interface{}{restdb.IDField: ID}); err != nil {
		return false, fmt.Errorf("Exist id:%s failed:%s ", ID, err.Error())
	} else {
		return exist, nil
	}

	tx.Commit()

	return false, nil
}

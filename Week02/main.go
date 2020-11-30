package main

import (
	"database/sql"
	"github.com/pkg/errors"
	"log"
)

// business --> service --> dao
var errMsg = map[error]string{
	sql.ErrNoRows: "no such data",
}

func dao(id int) (int, error) {
	if id < 0 {
		return 0, errors.Wrapf(sql.ErrNoRows, "cannot find any data with id  >>>>  [ %v ]  <<<<<", id)
	}
	return id, nil
}

func service(id int) (int, error) {
	return dao(id)
}

func business(id int) string {
	var (
		res int
		err error
	)

	if res, err = service(id); err != nil {
		// 1. 打印详细的堆栈日志
		// 2. 将提示信息返回给最终调用者
		log.Println("err with +v below")
		log.Printf("%+v", err)
		return errMsg[errors.Cause(err)]
	}

	log.Printf("result is %d", res)
	return "ok"
}

func main() {
	log.Println(business(1))
	log.Println("----------------------")
	log.Println(business(-1))
}

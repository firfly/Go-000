//+build wireinject 忽略编译

package main

import (
	"Week04/internal/biz"
	"Week04/internal/data"

	"github.com/google/wire"
)

func InitUserUsecase() *biz.UserUsecase {
	wire.Build(biz.NewUserUsecase, data.NewUserRepo)
	return &biz.UserUsecase{}
}
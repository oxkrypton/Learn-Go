package main

import (
	"fmt"
	"os"
	"runtime/trace"
)

func main(){
	//1.创建trace文件
	f,err:=os.Create("trace.out")
	if err!=nil{
		panic(err)
	}

	defer f.Close()

	//2.启动trace
	err=trace.Start(f)
	if err!=nil{
		panic(err)
	}

	//正常要调试的业务
	fmt.Println("hello GMP")

	//4.停止trace
	trace.Stop()
}
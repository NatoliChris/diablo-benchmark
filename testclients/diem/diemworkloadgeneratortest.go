package main

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs/parsers"
	//"diablo-benchmark/core/configs/parsers"
	"fmt"
)

func main(){
	//fmt.Printf("hello world")
	bc, err := parsers.ParseBenchConfig("configurations/workloads/diem/DiabloDiemBasic.yaml")
	if err != nil {
		panic(err)
	}
	var generator workloadgenerators.WorkloadGenerator
	intermediate := workloadgenerators.DiemWorkloadGenerator{}
	generator = intermediate.NewGenerator(nil, bc)
	err = generator.InitParams()
	if err != nil {
		panic(err)
	}
	generateWorkload, err := generator.GenerateWorkload()
	if err != nil {
		panic(err)
	}
	fmt.Println(generateWorkload)
}

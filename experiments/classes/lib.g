package lib

##include<stdio.h>

external func printf(format: string, a0: int)

func print(val: string, format: string = "%s\n"): void {
    printf(format: format, a0: val)
}
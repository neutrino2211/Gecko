package Main

import std.array

##include<stdio.h>

external func printf(format: string = "%u\n", val: int)

func Main() {
    // arrayOne: [int] = [0, 1, 2, 3, 4]
    listOne: std.array.List
    listOne = listOne.constructor(list_type_size: 4)
    listOne.push(item: 2)
    //printf(val: "Printing 0 to 4", format: "%s\n")
    printf(val: listOne.length)
    for value: int of [0, 1, 2, 3, 4]{
        printf(format: "Value: %u\n", val: value)
    }

    //for index: int in [0, 1, 2, 3, 4] {
    //    printf(format: "Index: %u\n", val: index)
    //}
}
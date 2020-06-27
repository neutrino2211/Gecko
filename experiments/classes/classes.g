package Main

import lib

class FancyType {
    __ctype__: string = "fancy_t"
    my_variable: int = 2
    
    func constructor(self: FancyType, my_variable: int = 2): FancyType {
        lib.print(val: self.my_variable, format: "My Variable %u\n")
        self.my_variable = my_variable
        return self
    }

    func point(self: FancyType, index: int): int {
        return index
    }
}

func returnsString(): string {
    return "Hello World!!"
}

func Main(argc: int, argv: [string]): int {

    // Variable declaration
    typeTest: FancyType

    // Contructor [need to automate this]
    typeTest = typeTest.constructor()

    // Method calling
    point: int = typeTest.point(index: 22)

    // Printing
    lib.print(val: "Printing Time!!\n\n")
    lib.print(format: "My Variable dereferencing => %u\n", val: typeTest.my_variable)
    lib.print(format:"Gecko version: %s\n", val: returnsString())
    lib.print(format: "Point => %u\n", val: point)

    return 2
}
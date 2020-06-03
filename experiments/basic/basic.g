package Main

import definitions

func Main(): void {
    result: int! = greeting(a0: "World!")
    printf(format: "Result is %u\n", a0: result)
}
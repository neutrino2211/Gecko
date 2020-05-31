package Main

##include <stdio.h>;

external func printf(format: string, a0: string): void {}

func Main() : void {
    printf(format: "Hello %s\n", a0: "World!")
}
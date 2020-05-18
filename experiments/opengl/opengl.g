package Main

# #include "cube.h"

external func glutMainLoop(): void {}
external func setAppName(name: String): void {}

func opengl_g(): void {
    @glutMainLoop()
}

func set_application_name(): void {
    @setAppName(name: "3d Cube")
}
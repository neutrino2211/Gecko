package Main

#int load();
#void setup();
#int create_window(char* name);
external func create_window(name: string): int {}
external func setup(): void {}
external func load(): int {}

func Main(): void {
    setup()
    create_window("Gecko + Imgui + Glfw + OpenGL")
    load()
}
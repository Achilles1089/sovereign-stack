import SwiftUI

@main
struct SovereignApp: App {
    @StateObject private var server = ServerManager()
    
    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(server)
                .onAppear {
                    server.start()
                }
        }
    }
}

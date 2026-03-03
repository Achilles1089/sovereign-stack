import SwiftUI

@main
struct SovereignApp: App {
    @StateObject private var server = ServerManager()
    
    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(server)
                .onAppear {
                    // Delay server start slightly to let UI settle
                    DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                        server.start()
                    }
                }
        }
    }
}

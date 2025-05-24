# Starship Game

I wanted to brush up on my Go skills so I built a fast-paced space shooter game that gets progressively more challenging.

**[Play the game here!](https://niknahadb.github.io/starship)**

## How to Play

- **↑/↓ Arrow Keys**: Thrust forward/backward
- **←/→ Arrow Keys**: Rotate your ship
- **Space**: Shoot bullets / Restart after game over
- **Goal**: Destroy meteors and survive as long as possible!

The game gets progressively harder - meteors spawn faster and move quicker as your score increases.

## Key Learnings and Takeaways

This project reinforced my appreciation for Go’s strengths in systems-level programming and real time application design. Embedding game assets with the `embed` package not only simplified distribution packaging images, audio, and fonts directly into the binary, but also removed runtime I/O complexity, letting me focus on core gameplay logic rather than file path management.

The custom `Vector`, `Timer`, and `Rect` types showcase how Go’s struct‐based composition and method receivers enable clear, reusable abstractions. Implementing 2D vector normalization and length calculations from first principles underscored the importance of precise math in physics simulations, while the `Timer` abstraction neatly decouples frame-based tick counting from higher-level spawn and cooldown logic. Likewise, the axis‐aligned bounding‐box collision detection in `Rect.Intersects` proved both performant and easy to reason about, ensuring accurate hit detection without introducing costly spatial partitioning.

In the game loop, integrating `ebiten.TPS()` with my `Timer` and player‐difficulty scaling demonstrated how to leverage Go’s predictable runtime for consistent behavior across platforms. Adjusting spawn rates and movement speeds based on `difficultyLevel` highlights a data‐driven approach to balancing. The `Update` and `Draw` methods adhere to Ebiten’s idiomatic patterns, blending Go’s concurrency‐safe design with straightforward, single threaded rendering.

Overall, this project made me appreicate Go's explicit error handling and minimal external dependencies as I architected an efficient program and leveraged Go’s tooling for cross-platform distribution in both desktop and WebAssembly contexts.

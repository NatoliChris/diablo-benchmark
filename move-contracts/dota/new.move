script {
  use Owner::Game;

  fun main(s: signer) {
    Game::new(&s, 10, 250);
  }
}

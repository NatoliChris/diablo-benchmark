script {
  use Owner::Game;

  fun main(a: address) {
    Game::update(a);
  }
}

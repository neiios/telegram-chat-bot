{
  lib,
  buildGo127Module,
  sqlc,
  version ? "0-unstable",
}:

buildGo127Module (finalAttrs: {
  pname = "telegram-chat-bot";
  inherit version;

  src = lib.fileset.toSource {
    root = ./.;
    fileset = lib.fileset.unions [
      (lib.fileset.fileFilter (file: file.hasExt "go") ./.)
      ./go.mod
      ./go.sum
      ./queries.sql
      ./schema.sql
      ./sqlc.yaml
    ];
  };

  vendorHash = "sha256-UTkp3qXSpq/hljlAh4CWMhg4T0r7yJwDR/CPWqhtNe4=";

  nativeBuildInputs = [ sqlc ];

  env.CGO_ENABLED = 0;

  postPatch = ''
    sqlc generate
  '';

  passthru.overrideModAttrs = _: {
    name = "telegram-chat-bot-go-modules";
  };

  ldflags = [
    "-s"
    "-w"
  ];

  meta = {
    description = "Telegram bot for running a daily roulette game in group chats";
    homepage = "https://github.com/neiios/telegram-chat-bot";
    license = lib.licenses.agpl3Only;
    mainProgram = "telegram-chat-bot";
    platforms = lib.platforms.unix;
  };
})

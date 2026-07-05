INSERT INTO eggs (category, name, description, docker_image, startup_command, stop_command)
SELECT 'generic', 'Custom Docker Container', 'Bring your own image and startup command.', 'ubuntu:22.04', 'tail -f /dev/null', 'exit'
WHERE NOT EXISTS (SELECT 1 FROM eggs WHERE name = 'Custom Docker Container');

INSERT INTO eggs (category, name, description, docker_image, startup_command, stop_command)
SELECT 'game', 'Minecraft: Vanilla', 'Official Minecraft server (itzg/minecraft-server). Set EULA=TRUE in environment.', 'itzg/minecraft-server', '', 'stop'
WHERE NOT EXISTS (SELECT 1 FROM eggs WHERE name = 'Minecraft: Vanilla');

#!/usr/bin/env node
'use strict';

const command = process.argv[2];

switch (command) {
  case 'collect':
    require('./collect.js')();
    break;
  default:
    console.log(
      'tracelet-next <command>\n\nCommands:\n  collect   Collect Next.js route stats into .tracelet/stats.json'
    );
    process.exit(command ? 1 : 0);
}

import { configuration } from 'app/wailsjs/go/models';
import Connection = configuration.RegisteredServer;

export interface RegisteredServerNode extends Connection {
  children: RegisteredServerNode[];
  header: string;
  showButtons: boolean;
}

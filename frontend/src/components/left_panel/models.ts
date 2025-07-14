import {configuration} from 'app/wailsjs/go/models';
import Connection = configuration.RegisteredServer;

export class RegisteredServerNode extends Connection {
  children: RegisteredServerNode[] = []
}

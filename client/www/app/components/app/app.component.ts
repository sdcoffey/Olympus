import {Component} from 'angular2/core';
import {FileListComponent} from '../filelist/filelist.component';
import {ApiClient} from '../../services/apiclient';
import {HTTP_PROVIDERS} from 'angular2/http';

@Component({
  selector: 'app',
  templateUrl: 'app/components/main/main.html',
  directives: [FileListComponent],
  providers: [ApiClient, HTTP_PROVIDERS]
})
export class AppComponent {}

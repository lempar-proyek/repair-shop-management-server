import { Datastore } from '@google-cloud/datastore';
import { Module } from '@nestjs/common';

const datastore = new Datastore()

const datastoreFactory = {
    provide: Datastore,
    useValue: datastore
}

@Module({
    providers: [
        datastoreFactory
    ],
    exports: [
        datastoreFactory
    ]
})
export class DatastoreModule { }

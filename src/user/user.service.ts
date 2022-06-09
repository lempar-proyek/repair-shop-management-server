import { Datastore } from '@google-cloud/datastore';
import { Injectable } from '@nestjs/common';
import { User } from './user.model';

@Injectable()
export class UserService {
    private kind = 'User'

    constructor(private datastore: Datastore) { }

    private datastoreToUser(entity: any): User {
        const user: User = {...entity}

        user.id = entity[this.datastore.KEY].name
        delete user[this.datastore.KEY]

        return user
    }

    async getUserByGoogleId(googleId: string): Promise<User | null> {
        const query = this.datastore.createQuery(this.kind).filter('googleId', googleId)
        const [users] = await this.datastore.runQuery(query)
        if(users.length == 0) {
            return null
        }
        const user = users[0];
        return this.datastoreToUser(user)
    }

    async createUser() {
        const userKey = this.datastore.key([this.kind, '1212121'])
        const user = {
            key: userKey,
            data: {
                name: 'Dummy',
                password: 'Ini user'
            }
        }
        await this.datastore.save(user)
        console.log('User saved');

    }
}
